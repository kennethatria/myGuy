package main

import (
	"log"
	"os"

	"store-service/internal/api/handlers"
	"store-service/internal/middleware"
	"store-service/internal/models"
	"store-service/internal/repositories"
	"store-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Database connection
	dbConnection := os.Getenv("DB_CONNECTION")
	if dbConnection == "" {
		log.Fatal("DB_CONNECTION environment variable is required")
	}

	db, err := gorm.Open(postgres.Open(dbConnection), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate database
	if err := db.AutoMigrate(&models.StoreItem{}, &models.Bid{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize repositories
	itemRepo := repositories.NewStoreItemRepository(db)
	bidRepo := repositories.NewBidRepository(db)

	// Initialize services
	storeService := services.NewStoreService(itemRepo, bidRepo)

	// Initialize handlers
	storeHandler := handlers.NewStoreHandler(storeService)

	// Initialize middleware
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	jwtMiddleware := middleware.NewJWTAuthMiddleware(jwtSecret)

	// Setup routes
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	api := router.Group("/api/v1")
	{
		// Public routes
		api.GET("/items", storeHandler.GetItems)
		api.GET("/items/:id", storeHandler.GetItem)
		api.GET("/items/:id/bids", storeHandler.GetItemBids)

		// Protected routes
		auth := api.Group("/")
		auth.Use(jwtMiddleware.AuthRequired())
		{
			// Item management
			auth.POST("/items", storeHandler.CreateItem)
			auth.PUT("/items/:id", storeHandler.UpdateItem)
			auth.DELETE("/items/:id", storeHandler.DeleteItem)
			auth.POST("/items/:id/purchase", storeHandler.PurchaseItem)

			// Bidding
			auth.POST("/items/:id/bids", storeHandler.PlaceBid)
			auth.POST("/items/:id/bids/:bidId/accept", storeHandler.AcceptBid)

			// User specific endpoints
			auth.GET("/user/listings", storeHandler.GetUserListings)
			auth.GET("/user/purchases", storeHandler.GetUserPurchases)
			auth.GET("/user/bids", storeHandler.GetUserBids)
		}
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Store service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}