package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/joho/godotenv"
	"github.com/slidingsoul/snap-link/controller"
	"github.com/slidingsoul/snap-link/internal/auth"
	"github.com/slidingsoul/snap-link/internal/ratelimit"
	"github.com/slidingsoul/snap-link/middleware"
	"github.com/slidingsoul/snap-link/models"
	"github.com/slidingsoul/snap-link/repository"
	"github.com/slidingsoul/snap-link/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	// Initialize database connection
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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

	// NOTE: AutoMigrate is convenient for this small service but has no
	// versioning/rollback. For larger prod schemas use golang-migrate/atlas.
	// NOTE: a legacy `lookups` table may exist in the DB; it is no longer used.
	if err := db.AutoMigrate(&models.Link{}); err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	log.Println("Database migration completed successfully")

	// Initialize repository, service, and controller layers
	linkRepo := repository.NewLinkRepository(db)
	linkService := service.NewLinkService(linkRepo)
	linkController := controller.NewLinkController(linkService)

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

	// Fixed per-IP rate limiter for /snap (see internal/ratelimit).
	rateLimiter := middleware.NewLimiter(ratelimit.DefaultPerMinute)

	// Routes — order matters: /health must be registered
	// before /:shortCode so the wildcard doesn't swallow it.
	// POST /snap requires a Neon JWT (Authorization: Bearer);
	// reads stay public.
	app.Get("/health", linkController.HealthCheck)
	app.Post("/snap", rateLimiter.Handler(), middleware.RequireAuth(authVerifier), linkController.CreateShortLink)

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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = shutdownCtx // fiber Shutdown has its own timeout below
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}
	log.Println("Server stopped")
}
