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
	"myguy/internal/chatnotify"
	"myguy/internal/proximity"
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
	shutdown, err := tracing.InitTracer(context.Background(), "myguy-backend")
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
	notifier := newTaskNotifier()
	// A nil *Notifier must reach the service as a nil interface (no chat)
	var taskNotifier services.TaskNotifier
	if notifier != nil {
		taskNotifier = notifier
	}
	taskService := services.NewTaskService(taskRepo, applicationRepo, taskNotifier).WithLocator(newLocator()).WithDistancer(newDistancer())
	reviewService := services.NewReviewService(reviewRepo, taskRepo, userRepo)

	go expireStaleTasks(taskService)
	if notifier != nil {
		go unlockMatchedChats(taskService, notifier)
	}

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
		auth.GET("/user/applications", handler.GetUserApplications)


		// Review routes
		auth.POST("/tasks/:id/reviews", handler.CreateReview)
		auth.GET("/tasks/:id/reviews/mine", handler.GetMyTaskReview)
		auth.GET("/users/:id/reviews", handler.GetUserReviews)
		auth.GET("/user/reviews", handler.GetMyReviews)
		auth.GET("/users/:id/network", handler.GetUserNetwork)
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

// expireStaleTasks marks gigs that got no application within their 24 hours
// as expired, checking every minute. Applying also checks the deadline, so
// the gap between runs can't let a late application in.
func expireStaleTasks(taskService *services.TaskService) {
	for range time.Tick(time.Minute) {
		n, err := taskService.ExpireStaleTasks(context.Background())
		if err != nil {
			log.Println("WARNING: expiring stale tasks failed:", err)
		} else if n > 0 {
			log.Printf("expired %d task(s) with no applications", n)
		}
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

// newDistancer sorts and tags gigs by distance through the proximity
// service, unless PROXIMITY_SORT_ENABLED is "false" (the rollback switch) or
// the proximity service isn't configured.
func newDistancer() services.Distancer {
	url, apiKey := os.Getenv("PROXIMITY_URL"), os.Getenv("INTERNAL_API_KEY")
	if os.Getenv("PROXIMITY_SORT_ENABLED") == "false" || url == "" || apiKey == "" {
		log.Println("distance sorting is off")
		return nil
	}
	return proximity.New(url, apiKey)
}

// newLocator saves gigs' rough locations in the proximity service, or
// returns nil (locations aren't saved) when PROXIMITY_URL or
// INTERNAL_API_KEY is unset.
func newLocator() services.Locator {
	url, apiKey := os.Getenv("PROXIMITY_URL"), os.Getenv("INTERNAL_API_KEY")
	if url == "" || apiKey == "" {
		log.Println("WARNING: PROXIMITY_URL or INTERNAL_API_KEY not set; gig locations won't be saved")
		return nil
	}
	return proximity.New(url, apiKey)
}

// unlockMatchedChats lets people matched before chat recorded matches keep
// talking: gig chats are locked until the poster accepts. Retries while the
// chat service starts up.
func unlockMatchedChats(taskService *services.TaskService, unlocker services.ChatUnlocker) {
	for attempt := 1; attempt <= 5; attempt++ {
		n, err := taskService.UnlockMatchedChats(context.Background(), unlocker)
		if err == nil {
			log.Printf("chat: %d matched pairs recorded", n)
			return
		}
		log.Printf("chat: recording matched pairs failed (attempt %d): %v", attempt, err)
		time.Sleep(time.Duration(attempt) * 10 * time.Second)
	}
}

// newTaskNotifier posts task events (applications, decisions) into Messages
// via the chat service, or does nothing if INTERNAL_API_KEY is unset.
func newTaskNotifier() *chatnotify.Notifier {
	apiKey := os.Getenv("INTERNAL_API_KEY")
	if apiKey == "" {
		log.Println("WARNING: INTERNAL_API_KEY not set; task events won't be posted to Messages")
		return nil
	}
	chatAPIURL := os.Getenv("CHAT_API_URL")
	if chatAPIURL == "" {
		chatAPIURL = "http://localhost:8082/api/v1"
	}
	return chatnotify.New(chatAPIURL, apiKey)
}

