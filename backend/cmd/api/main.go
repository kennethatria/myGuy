package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"myguy/internal/api"
	"myguy/internal/mailer"
	"myguy/internal/middleware"
	"myguy/internal/models"
	"myguy/internal/repositories"
	"myguy/internal/services"
	"myguy/internal/tracing"
)

func main() {
	// Load environment variables from .env file if it exists
	godotenv.Load() // Ignore error if .env doesn't exist

	// Initialize OpenTelemetry tracing
	zipkinURL := os.Getenv("ZIPKIN_URL")
	if zipkinURL == "" {
		zipkinURL = "http://localhost:9411/api/v2/spans"
	}
	shutdown, err := tracing.InitTracer("myguy-backend", zipkinURL)
	if err != nil {
		log.Fatal("Failed to initialize tracer:", err)
	}
	defer shutdown(context.Background())

	// Initialize database
	db, err := gorm.Open(postgres.Open(os.Getenv("DB_CONNECTION")), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate database
	err = db.AutoMigrate(
		&models.User{},
		&models.Task{},
		&models.Application{},
		&models.Review{},
		&models.LoginCode{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	// Initialize repositories
	userRepo := repositories.NewGormUserRepository(db)
	if err := userRepo.RecalculateAllRatings(context.Background()); err != nil {
		log.Println("WARNING: failed to recalculate user ratings:", err)
	}
	taskRepo := repositories.NewGormTaskRepository(db)
	applicationRepo := repositories.NewGormApplicationRepository(db)
	reviewRepo := repositories.NewGormReviewRepository(db)
	loginCodeRepo := repositories.NewGormLoginCodeRepository(db)

	// Initialize services
	userService := services.NewUserService(userRepo)
	taskService := services.NewTaskService(taskRepo, applicationRepo)
	reviewService := services.NewReviewService(reviewRepo, taskRepo, userRepo)

	authService := services.NewAuthService(userRepo, loginCodeRepo, newCodeSender(), os.Getenv("JWT_SECRET"))

	// Initialize JWT middleware
	jwtMiddleware := middleware.NewJWTAuthMiddleware(os.Getenv("JWT_SECRET"))
	// Initialize handlers
	handler := api.NewHandler(authService, userService, taskService, reviewService, jwtMiddleware)

	// Setup router
	r := gin.Default()
	r.Use(otelgin.Middleware("myguy-backend"))

	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Public routes
	r.GET("/api/v1/time", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"time": time.Now().Format(time.RFC3339),
		})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/api/v1/server-time", handler.GetServerTime)
	r.POST("/api/v1/auth/request-code", handler.RequestLoginCode)
	r.POST("/api/v1/auth/verify-code", handler.VerifyLoginCode)
	r.POST("/api/v1/auth/complete-signup", handler.CompleteSignup)
	// Protected routes
	auth := r.Group("/api/v1")
	auth.Use(jwtMiddleware.AuthRequired())
	{
		// Task routes
		auth.POST("/tasks", handler.CreateTask)
		auth.GET("/tasks", handler.ListTasks)
		auth.GET("/tasks/:id", handler.GetTask)
		auth.PUT("/tasks/:id", handler.UpdateTask)
		auth.PATCH("/tasks/:id/status", handler.UpdateTaskStatus)
		auth.DELETE("/tasks/:id", handler.DeleteTask)
		auth.POST("/tasks/:id/apply", handler.ApplyForTask)
		auth.GET("/tasks/:id/applications", handler.GetTaskApplications)
		auth.PATCH("/tasks/:id/applications/:applicationId", handler.RespondToApplication)

		// User-specific task routes
		auth.GET("/user/tasks", handler.GetUserTasks)
		auth.GET("/user/tasks/assigned", handler.GetAssignedTasks)


		// Review routes
		auth.POST("/tasks/:id/reviews", handler.CreateReview)
		auth.GET("/tasks/:id/reviews/mine", handler.GetMyTaskReview)
		auth.GET("/users/:id/reviews", handler.GetUserReviews)
		// User routes
		auth.GET("/users/:id", handler.GetUserByID)

		// Application chat authorization (used by the chat service)
		auth.GET("/applications/:id/participants", handler.GetApplicationParticipants)

		// Profile routes
		auth.GET("/profile", handler.GetProfile)
		auth.PUT("/profile", handler.UpdateProfile)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// newCodeSender emails login codes over SMTP, or logs them when SMTP_HOST is
// unset so local development works without a mail account.
func newCodeSender() services.CodeSender {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		log.Println("WARNING: SMTP_HOST not set; login codes will be logged, not emailed")
		return mailer.LogSender{}
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return mailer.NewSMTPSender(mailer.SMTPConfig{
		Host:     host,
		Port:     port,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
	})
}
