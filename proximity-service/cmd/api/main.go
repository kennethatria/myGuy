package main

import (
	"context"
	"log"
	"os"
	"time"

	"proximity-service/internal/api/handlers"
	"proximity-service/internal/middleware"
	"proximity-service/internal/repositories"
	"proximity-service/internal/services"
	"proximity-service/internal/tracing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	apiKey := os.Getenv("INTERNAL_API_KEY")
	if apiKey == "" {
		log.Fatal("INTERNAL_API_KEY environment variable is required")
	}

	// Initialize OpenTelemetry tracing
	shutdown, err := tracing.InitTracer(context.Background(), "myguy-proximity-service")
	if err != nil {
		log.Fatal("Failed to initialize tracer:", err)
	}
	defer shutdown(context.Background())

	rdb := redis.NewClient(&redis.Options{
		Addr:     getenv("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	service := services.NewProximityService(repositories.NewRedisLocationRepository(rdb))
	go cleanupDaily(service)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), otelgin.Middleware("myguy-proximity-service"))
	handlers.NewHandler(service).Register(router, middleware.InternalKey(apiKey))

	port := getenv("PORT", "8083")
	log.Printf("Proximity service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// cleanupDaily removes locations saved more than 30 days ago, at start-up
// and then once a day.
func cleanupDaily(service *services.ProximityService) {
	for {
		if n, err := service.Cleanup(context.Background()); err != nil {
			log.Println("WARNING: location cleanup failed:", err)
		} else if n > 0 {
			log.Printf("removed %d old location(s)", n)
		}
		time.Sleep(24 * time.Hour)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
