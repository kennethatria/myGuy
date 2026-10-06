package main

import (
	"context"
	"log"
	"os"
	"time"

	"store-service/internal/api/handlers"
	"store-service/internal/media"
	"store-service/internal/middleware"
	"store-service/internal/models"
	"store-service/internal/proximity"
	"store-service/internal/repositories"
	"store-service/internal/services"
	"store-service/internal/tracing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize OpenTelemetry tracing
	zipkinURL := os.Getenv("ZIPKIN_URL")
	if zipkinURL == "" {
		zipkinURL = "http://localhost:9411/api/v2/spans"
	}
	shutdown, err := tracing.InitTracer("myguy-store-service", zipkinURL)
	if err != nil {
		log.Fatal("Failed to initialize tracer:", err)
	}
	defer shutdown(context.Background())

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
	if err := db.AutoMigrate(&models.StoreItem{}, &models.ItemImage{}, &models.Bid{}, &models.BookingRequest{}, &models.User{}, &models.ItemRequest{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize repositories
	itemRepo := repositories.NewStoreItemRepository(db)
	bidRepo := repositories.NewBidRepository(db)
	bookingRepo := repositories.NewBookingRequestRepository(db)
	userRepo := repositories.NewUserRepository(db)
	requestRepo := repositories.NewItemRequestRepository(db)

	// Initialize services
	storeService := services.NewStoreService(db, itemRepo, bidRepo, bookingRepo, userRepo).
		WithRequests(requestRepo, services.NewHTTPChatNotifier()).
		WithLocator(newLocator()).
		WithDistancer(newDistancer())
	requestService := services.NewRequestService(requestRepo, itemRepo).
		WithLocator(newLocator()).
		WithDistancer(newDistancer())

	// Listings from before notes had deadlines get a fresh 24 hours
	if n, err := itemRepo.StartMissingDeadlines(time.Now().UTC().Add(services.ListingLifetime)); err != nil {
		log.Println("WARNING: starting listing deadlines failed:", err)
	} else if n > 0 {
		log.Printf("gave %d listing(s) a 24 hour deadline", n)
	}
	go expireStaleNotes(storeService, requestService)

	// Photos uploaded before uploads were cleaned may still carry EXIF
	// metadata such as GPS positions; clean them once, in place
	if n, err := media.CleanExisting(handlers.UploadsDir); err != nil {
		log.Println("WARNING: cleaning stored photos failed:", err)
	} else if n > 0 {
		log.Printf("removed metadata from %d stored photo(s)", n)
	}

	// Initialize handlers
	storeHandler := handlers.NewStoreHandler(storeService)
	requestHandler := handlers.NewRequestHandler(requestService)

	// Initialize middleware
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	jwtMiddleware := middleware.NewJWTAuthMiddleware(jwtSecret, userRepo)

	// Setup routes
	router := gin.Default()
	router.Use(otelgin.Middleware("myguy-store-service"))
	
	// Serve static files for uploaded images
	router.Static("/uploads", "./uploads")

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
		// Every route needs a signed-in user: listings and requests name
		// their sellers and requesters, which shouldn't be scrapeable
		auth := api.Group("/")
		auth.Use(jwtMiddleware.AuthRequired())
		{
			// Browsing
			auth.GET("/items", storeHandler.GetItems)
			auth.GET("/items/:id", storeHandler.GetItem)
			auth.GET("/items/:id/bids", storeHandler.GetItemBids)
			auth.GET("/requests", requestHandler.GetRequests)
			auth.GET("/requests/:id", requestHandler.GetRequest)
			auth.GET("/requests/:id/listings", requestHandler.GetRequestListings)

			// Item management
			auth.POST("/items", storeHandler.CreateItem)
			auth.PUT("/items/:id", storeHandler.UpdateItem)
			auth.DELETE("/items/:id", storeHandler.DeleteItem)
			auth.POST("/items/:id/repost", storeHandler.RepostItem)

			// Requests ("wanted" notes)
			auth.POST("/requests", requestHandler.CreateRequest)
			auth.POST("/requests/:id/repost", requestHandler.RepostRequest)
			auth.DELETE("/requests/:id", requestHandler.DeleteRequest)
			auth.GET("/user/requests", requestHandler.GetUserRequests)
			auth.POST("/items/:id/purchase", storeHandler.PurchaseItem)

			// Bidding
			auth.POST("/items/:id/bids", storeHandler.PlaceBid)
			auth.POST("/items/:id/bids/:bidId/accept", storeHandler.AcceptBid)

			// Booking requests
			auth.POST("/items/:id/booking-request", storeHandler.CreateBookingRequest)
			auth.GET("/items/:id/booking-request", storeHandler.GetBookingRequest)
			auth.GET("/items/:id/booking-requests", storeHandler.GetAllBookingRequests)
			auth.POST("/booking-requests/:requestId/approve", storeHandler.ApproveBookingRequest)
			auth.POST("/booking-requests/:requestId/reject", storeHandler.RejectBookingRequest)
			auth.POST("/booking-requests/:requestId/release", storeHandler.ReleaseBooking)
			auth.POST("/booking-requests/:requestId/confirm-received", storeHandler.ConfirmItemReceived)
			auth.POST("/booking-requests/:requestId/confirm-delivery", storeHandler.ConfirmDelivery)
			auth.POST("/booking-requests/:requestId/rate-seller", storeHandler.SubmitBuyerRating)
			auth.POST("/booking-requests/:requestId/rate-buyer", storeHandler.SubmitSellerRating)

			// User specific endpoints
			auth.GET("/user/listings", storeHandler.GetUserListings)
			auth.GET("/user/purchases", storeHandler.GetUserPurchases)
			auth.GET("/user/bids", storeHandler.GetUserBids)
			auth.GET("/user/booking-requests", storeHandler.GetUserBookingRequests)
			auth.GET("/users/:id/ratings", storeHandler.GetUserRatings)
			auth.GET("/user/ratings", storeHandler.GetMyRatings)
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
// newDistancer sorts and tags listings and requests by distance through
// the proximity service, unless PROXIMITY_SORT_ENABLED is "false" (the
// rollback switch) or the proximity service isn't configured.
func newDistancer() services.Distancer {
	url, apiKey := os.Getenv("PROXIMITY_URL"), os.Getenv("INTERNAL_API_KEY")
	if os.Getenv("PROXIMITY_SORT_ENABLED") == "false" || url == "" || apiKey == "" {
		log.Println("distance sorting is off")
		return nil
	}
	return proximity.New(url, apiKey)
}

// newLocator saves rough locations in the proximity service, or returns
// nil (locations aren't saved) when PROXIMITY_URL or INTERNAL_API_KEY is
// unset.
func newLocator() services.Locator {
	url, apiKey := os.Getenv("PROXIMITY_URL"), os.Getenv("INTERNAL_API_KEY")
	if url == "" || apiKey == "" {
		log.Println("WARNING: PROXIMITY_URL or INTERNAL_API_KEY not set; locations won't be saved")
		return nil
	}
	return proximity.New(url, apiKey)
}

// expireStaleNotes takes listings and requests that got no reaction within
// their 24 hours off the board, checking every minute. Booking also checks a
// listing's deadline, so the gap between runs can't let a late request in.
func expireStaleNotes(storeService *services.StoreService, requestService *services.RequestService) {
	for range time.Tick(time.Minute) {
		if n, err := storeService.ExpireStaleItems(); err != nil {
			log.Println("WARNING: expiring stale listings failed:", err)
		} else if n > 0 {
			log.Printf("expired %d listing(s) with no bids or booking requests", n)
		}
		if n, err := requestService.ExpireStaleRequests(); err != nil {
			log.Println("WARNING: expiring stale requests failed:", err)
		} else if n > 0 {
			log.Printf("expired %d request(s) with no listings", n)
		}
	}
}
