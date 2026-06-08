package main

import (
	"log"
	"order-service/internal/handler"
	"order-service/internal/models"
	"order-service/internal/repository"
	"order-service/internal/service"
	"order-service/pkg/config"
	"order-service/pkg/db"
	"order-service/pkg/logger"

	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// Get database configuration from environment
	logger.Init()
	defer logger.Sync()

	// Load environment variables from .env file
	if err := godotenv.Load(".env"); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}
	dsn := os.Getenv("DATABASE_DSN")

	gormDB, err := db.InitDB(dsn)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}

	if err := gormDB.AutoMigrate(&models.Order{}, &models.OrderItem{}); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	paymentURL := config.GetEnv(
		"PAYMENT_SERVICE_URL",
		"http://localhost:8085",
	)
	repo := repository.NewOrderRepository(gormDB)
	svc := service.NewOrderService(repo, paymentURL)
	h := handler.NewOrderHandler(svc)

	r := gin.Default()

	api := r.Group("orders")
	{
		api.POST("", h.Create)
		api.GET("/:id", h.GetOrder)
		api.PATCH("/:id/update-status", h.UpdateStatus)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}
	log.Printf("✅ OrderService running on port %s", port)
	r.Run(":" + port)
}
