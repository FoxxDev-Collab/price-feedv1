package main

import (
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

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// API routes
	api := app.Group("/api")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)
	auth.Post("/logout", h.Logout)
	auth.Get("/me", middleware.AuthRequired(cfg), h.GetCurrentUser)
	auth.Post("/refresh", middleware.AuthRequired(cfg), h.RefreshToken)

	// User routes (authenticated)
	users := api.Group("/users", middleware.AuthRequired(cfg))
	users.Get("/:id", h.GetUser)
	users.Put("/:id", h.UpdateUser)
	users.Post("/:id/change-password", h.ChangePassword)
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

	// Store routes (public read, authenticated write)
	stores := api.Group("/stores")
	stores.Get("/", h.ListStores)
	stores.Get("/stats", h.GetStoreStats)
	stores.Get("/search", h.SearchStores)
	stores.Get("/:id", h.GetStore)
	stores.Post("/", middleware.AuthRequired(cfg), h.UserCreateStore)
	stores.Put("/:id", middleware.AuthRequired(cfg), h.UserUpdateStore)
	stores.Delete("/:id", middleware.AuthRequired(cfg), h.UserDeleteStore)

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
	items.Post("/", middleware.AuthRequired(cfg), h.UserCreateItem)
	items.Put("/:id", middleware.AuthRequired(cfg), h.UserUpdateItem)
	items.Delete("/:id", middleware.AuthRequired(cfg), h.UserDeleteItem)

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
	prices.Post("/", middleware.AuthRequired(cfg), h.CreatePrice)
	prices.Post("/:id/verify", middleware.AuthRequired(cfg), h.VerifyPrice)
	prices.Put("/:id", middleware.AuthRequired(cfg), h.UserUpdatePrice)
	prices.Delete("/:id", middleware.AuthRequired(cfg), h.UserDeletePrice)

	// Admin price routes
	admin.Put("/prices/:id", h.UpdatePrice)
	admin.Delete("/prices/:id", h.DeletePrice)

	// Shopping list routes (authenticated)
	lists := api.Group("/lists", middleware.AuthRequired(cfg))
	lists.Get("/", h.ListShoppingLists)
	lists.Post("/", h.CreateShoppingList)
	lists.Get("/:id", h.GetShoppingList)
	lists.Put("/:id", h.UpdateShoppingList)
	lists.Delete("/:id", h.DeleteShoppingList)
	lists.Post("/:id/items", h.AddItemToList)
	lists.Put("/:id/items/:item_id", h.UpdateListItem)
	lists.Delete("/:id/items/:item_id", h.RemoveItemFromList)
	lists.Post("/:id/build-plan", h.BuildShoppingPlan)
	lists.Post("/:id/complete", h.CompleteShoppingList)
	lists.Post("/:id/reopen", h.ReopenShoppingList)
	lists.Post("/:id/duplicate", h.DuplicateShoppingList)

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
