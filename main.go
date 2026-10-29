package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/joho/godotenv"
	"github.com/slidingsoul/snap-link/controller"
	"github.com/slidingsoul/snap-link/internal/auth"
	"github.com/slidingsoul/snap-link/internal/ratelimit"
	"github.com/slidingsoul/snap-link/middleware"
	"github.com/slidingsoul/snap-link/repository"
	"github.com/slidingsoul/snap-link/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	// requestTimeout bounds the whole handler (including DB queries).
	requestTimeout = 5 * time.Second

	// Neon scales to zero, so the first connection after idle can take a
	// few seconds. Retry instead of failing the deploy/restart on a cold DB.
	dbConnectAttempts = 5
	dbConnectBaseWait = time.Second
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Load environment variables (defaults unified to 8081: see .env.example, Dockerfile, compose)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Neon Managed Better Auth base URL (per-branch, Console → Branch → Auth).
	// e.g. https://ep-xxx.aws.neon.tech/neondb/auth
	// JWKS + issuer are derived: <base>/.well-known/jwks.json, origin of <base>.
	authBaseURL := os.Getenv("NEON_AUTH_BASE_URL")
	if authBaseURL == "" {
		log.Fatal("NEON_AUTH_BASE_URL environment variable is required")
	}
	authVerifier, err := auth.NewVerifier(authBaseURL)
	if err != nil {
		log.Fatalf("Failed to init Neon auth verifier: %v", err)
	}
	log.Println("Neon auth verifier initialized")

	// Public origin used to build shareable links. Set it in production
	// (e.g. https://snap.example.com); otherwise it is derived per request.
	publicBaseURL := os.Getenv("PUBLIC_BASE_URL")
	if publicBaseURL == "" {
		log.Println("Warning: PUBLIC_BASE_URL not set — short links are built from the request Host/Proto")
	}

	// Initialize database connection (with retry for Neon cold starts)
	db, err := connectDB(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database after %d attempts: %v", dbConnectAttempts, err)
	}

	// Bound the pool so one deploy can't exhaust Postgres.
	// Why: GORM defaults to unlimited opens; under burst traffic
	// you get "too many connections" instead of graceful queueing.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
	} else {
		log.Printf("Warning: could not configure connection pool: %v", err)
	}

	log.Println("Database connection established successfully")

	// Schema is owned by migrations/ (golang-migrate, applied manually — see
	// README "Run Migrations"). The app never runs DDL itself; it only verifies
	// the expected tables exist so a missing migration fails fast with a
	// clear message instead of obscure query errors at request time.
	// NOTE: a legacy `lookups` table may exist in the DB; it is no longer used.
	var tableExists bool
	schemaCtx, schemaCancel := context.WithTimeout(context.Background(), requestTimeout)
	if err := db.WithContext(schemaCtx).Raw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'links')").Scan(&tableExists).Error; err != nil {
		schemaCancel()
		log.Fatalf("Failed to check schema: %v", err)
	}
	schemaCancel()
	if !tableExists {
		log.Fatal("links table not found: apply migrations/ to DATABASE_URL first (see README)")
	}

	// Initialize repository, service, and controller layers
	linkRepo := repository.NewLinkRepository(db)
	linkService := service.NewLinkService(linkRepo)
	linkController := controller.NewLinkController(linkService, publicBaseURL)

	// Initialize Fiber app with production-safe defaults.
	app := fiber.New(fiber.Config{
		AppName:      "Snap Link Service",
		ServerHeader: "", // don't leak implementation/version
		BodyLimit:    16 * 1024,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		// Behind Render/reverse proxy the client IP arrives in X-Forwarded-For.
		// Needed so per-IP rate limiting sees real clients, not the proxy.
		// NOTE: clients can spoof this header if they reach the app directly;
		// front with a proxy that overwrites it in that setup.
		ProxyHeader: "X-Forwarded-For",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(helmet.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency}) ${locals:requestid}\n",
	}))

	// CORS: the browser frontend is a different origin, and Authorization is
	// a non-simple header, so /snap needs an OPTIONS preflight. Empty config
	// means "no cross-origin allowed" (Fiber would otherwise default to "*").
	corsConfig := cors.Config{
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Authorization,Content-Type",
		MaxAge:       600,
	}
	if allowed := os.Getenv("CORS_ALLOWED_ORIGINS"); allowed != "" {
		corsConfig.AllowOrigins = allowed
	} else {
		log.Println("Warning: CORS_ALLOWED_ORIGINS not set — cross-origin browser requests will be rejected")
		corsConfig.AllowOriginsFunc = func(string) bool { return false }
	}
	app.Use(cors.New(corsConfig))

	// Give handlers (and DB queries) a deadline; see middleware/timeout.go.
	app.Use(middleware.Timeout(requestTimeout))

	// Fixed per-user rate limiter for /snap (see internal/ratelimit).
	rateLimiter := middleware.NewLimiter(ratelimit.DefaultPerMinute)

	// Routes — order matters: /health and /livez must be registered
	// before /:shortCode so the wildcard doesn't swallow them.
	// POST /snap: auth first, then rate limit, so the quota is keyed on the
	// JWT subject instead of a spoofable header.
	// Reads stay public.
	app.Get("/livez", linkController.Liveness)
	app.Get("/health", linkController.HealthCheck)
	app.Post("/snap", middleware.RequireAuth(authVerifier), rateLimiter.Handler(), linkController.CreateShortLink)

	app.Get("/:shortCode", linkController.GetOriginalURL)

	// Graceful shutdown: finish in-flight requests on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		serverAddr := fmt.Sprintf(":%s", port)
		log.Printf("Starting server on http://0.0.0.0:%s", port)
		if err := app.Listen(serverAddr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received, draining connections...")
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}
	log.Println("Server stopped")
}

// connectDB opens the pool and pings it, retrying with linear backoff.
// Neon scales to zero; without this a cold database turns a routine deploy
// into a failed boot.
func connectDB(databaseURL string) (*gorm.DB, error) {
	var lastErr error
	for attempt := 1; attempt <= dbConnectAttempts; attempt++ {
		db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
		if err == nil {
			var sqlDB *sql.DB
			if sqlDB, err = db.DB(); err == nil {
				pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				err = sqlDB.PingContext(pingCtx)
				cancel()
			}
			if err == nil {
				return db, nil
			}
		}

		lastErr = err
		if attempt == dbConnectAttempts {
			break
		}
		wait := time.Duration(attempt) * dbConnectBaseWait
		log.Printf("Database connect attempt %d/%d failed (%v); retrying in %s", attempt, dbConnectAttempts, err, wait)
		time.Sleep(wait)
	}
	return nil, lastErr
}
