package main

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/foxxcyber/price-feed/internal/config"
	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/handlers"
	"github.com/foxxcyber/price-feed/internal/middleware"
	"github.com/foxxcyber/price-feed/internal/services"
)

func main() {
	// Load .env file if it exists
	godotenv.Load()

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create admin user if it doesn't exist
	if err := database.EnsureAdminUser(db, cfg); err != nil {
		log.Printf("Warning: Could not ensure admin user: %v", err)
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: handlers.ErrorHandler,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Create handler with dependencies
	h := handlers.New(db, cfg)

	// Initialize Google Maps service and handler
	mapsService := services.NewGoogleMapsService(cfg.GoogleMapsAPIKey)
	mapsHandler := handlers.NewMapsHandler(mapsService, cfg.GoogleMapsAPIKey)

	// Initialize Email service and settings handler
	emailService := services.NewEmailService(db, cfg)
	settingsHandler := handlers.NewSettingsHandler(db, cfg, emailService)

	// Initialize Storage service for receipts (load from database settings)
	var receiptHandler *handlers.ReceiptHandler
	initReceiptService := func() {
		// Create encryption key from JWT secret
		encryptionKey := make([]byte, 32)
		copy(encryptionKey, []byte(cfg.JWTSecret))

		// Load S3 settings from database
		ctx := context.Background()
		settings, err := db.GetSettingsByCategoryAsMap(ctx, "storage", encryptionKey, true)
		if err != nil {
			log.Printf("Warning: Failed to load storage settings: %v", err)
			return
		}

		// Helper to get string from interface{}
		getString := func(key string) string {
			if v, ok := settings[key]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
			return ""
		}

		enabled := getString("s3_enabled") == "true"
		endpoint := getString("s3_endpoint")
		accessKey := getString("s3_access_key")
		secretKey := getString("s3_secret_key")
		bucket := getString("s3_bucket")
		region := getString("s3_region")
		useSSL := getString("s3_use_ssl") == "true"

		if !enabled {
			log.Println("Receipt storage is disabled in settings")
			return
		}

		if endpoint == "" || accessKey == "" || secretKey == "" {
			log.Println("S3 credentials not configured in settings, receipt scanning disabled")
			return
		}

		if bucket == "" {
			bucket = "receipts"
		}
		if region == "" {
			region = "garage"
		}

		storageService, err := services.NewStorageService(endpoint, accessKey, secretKey, bucket, region, useSSL)
		if err != nil {
			log.Printf("Warning: Failed to initialize storage service: %v", err)
			return
		}

		// Ensure bucket exists
		if err := storageService.EnsureBucket(ctx); err != nil {
			log.Printf("Warning: Failed to ensure S3 bucket exists: %v", err)
		}

		// Initialize OCR service
		ocrService, err := services.NewOCRService()
		if err != nil {
			log.Printf("Warning: Failed to initialize OCR service: %v", err)
			return
		}

		// Initialize receipt parser and matcher
		receiptParser := services.NewReceiptParser()
		itemMatcher := services.NewItemMatcher(db)

		// Create receipt handler
		receiptHandler = handlers.NewReceiptHandler(
			db, cfg, storageService, ocrService, receiptParser, itemMatcher,
		)
		log.Println("Receipt scanning service initialized")

		// Run cleanup of expired receipts on startup
		go func() {
			cleanupCtx := context.Background()
			keys, err := db.CleanupExpiredReceipts(cleanupCtx)
			if err != nil {
				log.Printf("Warning: Failed to cleanup expired receipts: %v", err)
				return
			}
			if len(keys) > 0 {
				log.Printf("Cleaned up %d expired receipt(s) from database", len(keys))
				// Delete S3 objects
				if err := storageService.DeleteMultiple(cleanupCtx, keys); err != nil {
					log.Printf("Warning: Failed to delete some S3 objects: %v", err)
				} else {
					log.Printf("Deleted %d expired receipt image(s) from storage", len(keys))
				}
			}
		}()
	}
	initReceiptService()

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// API routes
	api := app.Group("/api")

	// Create email verification middleware for write operations
	emailVerified := middleware.EmailVerifiedRequiredFunc(h.CreateEmailVerificationChecker())

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Get("/captcha-config", h.GetCaptchaConfig)
	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)
	auth.Post("/logout", h.Logout)
	auth.Get("/me", middleware.AuthRequired(cfg), h.GetCurrentUser)
	auth.Post("/refresh", middleware.AuthRequired(cfg), h.RefreshToken)
	auth.Get("/verify-email", h.VerifyEmail)
	auth.Post("/resend-verification", middleware.AuthRequired(cfg), h.ResendVerificationEmail)
	auth.Get("/verification-status", middleware.AuthRequired(cfg), h.GetEmailVerificationStatus)

	// User routes (authenticated)
	users := api.Group("/users", middleware.AuthRequired(cfg))
	users.Get("/:id", h.GetUser)
	users.Put("/:id", emailVerified, h.UpdateUser)
	users.Post("/:id/change-password", emailVerified, h.ChangePassword)
	users.Get("/:id/stats", h.GetUserStats)

	// Region routes (public read, admin write)
	regions := api.Group("/regions")
	regions.Get("/", h.ListRegions)
	regions.Get("/states", h.GetRegionStates)
	regions.Get("/stats", h.GetRegionStats)
	regions.Get("/search", h.SearchRegions)
	regions.Get("/:id", h.GetRegion)

	// Admin routes (admin only)
	admin := api.Group("/admin", middleware.AuthRequired(cfg), middleware.AdminRequired())
	admin.Post("/users", h.AdminCreateUser)
	admin.Get("/users", h.AdminListUsers)
	admin.Get("/users/:id", h.AdminGetUser)
	admin.Put("/users/:id", h.AdminUpdateUser)
	admin.Delete("/users/:id", h.AdminDeleteUser)
	admin.Get("/stats", h.AdminGetStats)

	// Admin region routes
	admin.Post("/regions", h.CreateRegion)
	admin.Put("/regions/:id", h.UpdateRegion)
	admin.Delete("/regions/:id", h.DeleteRegion)

	// Admin settings routes
	admin.Get("/settings", settingsHandler.GetAllSettings)
	admin.Get("/settings/:category", settingsHandler.GetSettingsByCategory)
	admin.Put("/settings/:category", settingsHandler.UpdateSettings)

	// Admin email routes
	admin.Get("/email/config", settingsHandler.GetEmailConfig)
	admin.Post("/email/test", settingsHandler.SendTestEmail)
	admin.Put("/email/config", settingsHandler.UpdateEmailSettings)
	admin.Get("/email/status", settingsHandler.GetEmailStatus)

	// Admin storage routes (S3/Garage)
	admin.Get("/storage/config", settingsHandler.GetStorageConfig)
	admin.Put("/storage/config", settingsHandler.UpdateStorageSettings)
	admin.Post("/storage/test", settingsHandler.TestStorageConnection)

	// Admin security routes
	admin.Post("/settings/regenerate-jwt-secret", settingsHandler.RegenerateJWTSecret)

	// Store routes (public read, authenticated write)
	stores := api.Group("/stores")
	stores.Get("/", h.ListStores)
	stores.Get("/stats", h.GetStoreStats)
	stores.Get("/search", h.SearchStores)
	stores.Get("/:id", h.GetStore)
	stores.Post("/", middleware.AuthRequired(cfg), emailVerified, h.UserCreateStore)
	stores.Put("/:id", middleware.AuthRequired(cfg), emailVerified, h.UserUpdateStore)
	stores.Delete("/:id", middleware.AuthRequired(cfg), emailVerified, h.UserDeleteStore)

	// Admin store routes
	admin.Post("/stores", h.CreateStore)
	admin.Put("/stores/:id", h.UpdateStore)
	admin.Delete("/stores/:id", h.DeleteStore)
	admin.Post("/stores/:id/verify", h.VerifyStore)

	// Item routes (public read, authenticated write)
	items := api.Group("/items")
	items.Get("/", h.ListItems)
	items.Get("/stats", h.GetItemStats)
	items.Get("/search", h.SearchItems)
	items.Get("/:id", h.GetItem)
	items.Post("/", middleware.AuthRequired(cfg), emailVerified, h.UserCreateItem)
	items.Put("/:id", middleware.AuthRequired(cfg), emailVerified, h.UserUpdateItem)
	items.Delete("/:id", middleware.AuthRequired(cfg), emailVerified, h.UserDeleteItem)

	// Tags routes (public)
	tags := api.Group("/tags")
	tags.Get("/", h.ListTags)

	// Admin item routes
	admin.Post("/items", h.CreateItem)
	admin.Put("/items/:id", h.UpdateItem)
	admin.Delete("/items/:id", h.DeleteItem)

	// Price routes (public read, authenticated write)
	prices := api.Group("/prices")
	prices.Get("/", h.ListPrices)
	prices.Get("/stats", h.GetPriceStats)
	prices.Get("/by-store/:store_id", h.GetPricesByStore)
	prices.Get("/by-item/:item_id", h.GetPricesByItem)
	prices.Get("/:id", h.GetPrice)
	prices.Post("/", middleware.AuthRequired(cfg), emailVerified, h.CreatePrice)
	prices.Post("/:id/verify", middleware.AuthRequired(cfg), emailVerified, h.VerifyPrice)
	prices.Put("/:id", middleware.AuthRequired(cfg), emailVerified, h.UserUpdatePrice)
	prices.Delete("/:id", middleware.AuthRequired(cfg), emailVerified, h.UserDeletePrice)

	// Admin price routes
	admin.Put("/prices/:id", h.UpdatePrice)
	admin.Delete("/prices/:id", h.DeletePrice)

	// Shopping list routes (authenticated, email verification required for write operations)
	lists := api.Group("/lists", middleware.AuthRequired(cfg))
	lists.Get("/", h.ListShoppingLists)
	lists.Post("/", emailVerified, h.CreateShoppingList)
	lists.Get("/:id", h.GetShoppingList)
	lists.Put("/:id", emailVerified, h.UpdateShoppingList)
	lists.Delete("/:id", emailVerified, h.DeleteShoppingList)
	lists.Post("/:id/items", emailVerified, h.AddItemToList)
	lists.Put("/:id/items/:item_id", emailVerified, h.UpdateListItem)
	lists.Delete("/:id/items/:item_id", emailVerified, h.RemoveItemFromList)
	lists.Post("/:id/build-plan", h.BuildShoppingPlan)
	lists.Post("/:id/complete", emailVerified, h.CompleteShoppingList)
	lists.Post("/:id/reopen", emailVerified, h.ReopenShoppingList)
	lists.Post("/:id/duplicate", emailVerified, h.DuplicateShoppingList)
	lists.Post("/:id/share", emailVerified, h.GenerateShareLink)
	lists.Post("/:id/email", emailVerified, h.EmailShoppingList)

	// Public share routes (no auth required)
	share := api.Group("/share")
	share.Get("/:token", h.GetSharedList)
	share.Post("/:token/items/:itemId/toggle", h.ToggleSharedListItem)

	// Receipt routes (authenticated, only if receipt handler is available)
	if receiptHandler != nil {
		receipts := api.Group("/receipts", middleware.AuthRequired(cfg))
		receipts.Post("/upload", emailVerified, receiptHandler.UploadReceipt)
		receipts.Get("/", receiptHandler.ListReceipts)
		receipts.Get("/:id", receiptHandler.GetReceipt)
		receipts.Put("/:id/items/:itemId", emailVerified, receiptHandler.UpdateReceiptItem)
		receipts.Post("/:id/confirm", emailVerified, receiptHandler.ConfirmReceipt)
		receipts.Delete("/:id", emailVerified, receiptHandler.DeleteReceipt)
		receipts.Get("/:id/image", receiptHandler.GetReceiptImage)
	}

	// Price comparison route (authenticated)
	api.Get("/compare", middleware.AuthRequired(cfg), h.GetPriceComparison)

	// Maps config route (public - needed for registration)
	api.Get("/maps/config", mapsHandler.GetConfig)

	// Maps routes (authenticated)
	maps := api.Group("/maps", middleware.AuthRequired(cfg))
	maps.Post("/geocode", mapsHandler.Geocode)
	maps.Post("/reverse-geocode", mapsHandler.ReverseGeocode)
	maps.Post("/nearby-stores", mapsHandler.NearbyStores)
	maps.Get("/place/:place_id", mapsHandler.GetPlaceDetails)

	// Shared list page route (serves the HTML page for shared lists)
	app.Get("/share/:token", func(c *fiber.Ctx) error {
		return c.SendFile("./web/share/index.html")
	})

	// Static files - serve the web/ directory
	app.Static("/", "./web", fiber.Static{
		Index:  "index.html",
		Browse: false,
	})

	// Fallback for SPA-style routing - serve index.html for unmatched routes
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/index.html")
	})

	// Get port from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
