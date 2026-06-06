package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/slidingsoul/snap-link/controller"
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

	// Load environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY environment variable is required")
	}

	// Initialize database connection
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection established successfully")

	// Auto-migrate the Link and Lookup models
	if err := db.AutoMigrate(&models.Link{}, &models.Lookup{}); err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	log.Println("Database migration completed successfully")

	// Initialize lookup repository and service
	lookupRepo := repository.NewLookupRepository(db)
	lookupService := service.NewLookupService(lookupRepo)

	// Seed initial lookup values if they don't exist
	if _, err := lookupService.GetLookupValue("LINK_PER_MINUTE"); err != nil {
		// Value doesn't exist, create it
		_, err := lookupService.UpdateLookupValue("LINK_PER_MINUTE", "5")
		if err != nil {
			log.Fatalf("Failed to seed LINK_PER_MINUTE lookup: %v", err)
		}
		log.Println("Initialized LINK_PER_MINUTE lookup to 5")
	}

	// Load rate limit from database
	rateLimit, err := lookupService.GetLookupValueAsInt("LINK_PER_MINUTE")
	if err != nil {
		log.Fatalf("Failed to load rate limit from database: %v", err)
	}
	log.Printf("Loaded rate limit from database: %d requests per minute", rateLimit)

	// Initialize repository, service, and controller layers
	linkRepo := repository.NewLinkRepository(db)
	linkService := service.NewLinkService(linkRepo)
	linkController := controller.NewLinkController(linkService, apiKey)
	lookupController := controller.NewLookupController(lookupService, apiKey)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Link Shortener Service",
		ServerHeader: "Link-Shortener/1.0",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path}\n",
	}))

	// Rate limiter for /snap endpoint with dynamic limit from database
	rateLimiter := limiter.New(limiter.Config{
		Max:        rateLimit,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fmt.Sprintf("Rate limit exceeded: %d requests per minute allowed", rateLimit),
			})
		},
	})

	// Routes
	app.Get("/health", linkController.HealthCheck)
	app.Post("/snap", rateLimiter, linkController.CreateShortLink)
	app.Get("/:shortCode", linkController.GetOriginalURL)

	// Config routes
	app.Put("/config/lookup", lookupController.UpdateLookup)
	app.Get("/config/lookup/:key", lookupController.GetLookup)

	// Start server
	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on http://0.0.0.0:%s", port)
	if err := app.Listen(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
