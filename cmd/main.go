package main

import (
	"log"
	"order_service/internal/handler"
	"order_service/internal/models"
	"order_service/internal/repository"
	"order_service/internal/service"

	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/miank1/ecommerce_backend/pkg/config"
	"github.com/miank1/ecommerce_backend/pkg/db"
	"github.com/miank1/ecommerce_backend/pkg/logger"
	"github.com/miank1/ecommerce_backend/pkg/rabbitmq"
)

func LoadEnv() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
}

func main() {

	// Get database configuration from environment
	logger.Init()
	defer logger.Sync()

	// Load environment variables from .env file
	LoadEnv()

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

	// RabbitMQ
	rabbit, err := rabbitmq.New(
		config.GetEnv("RABBITMQ_URL", ""),
	)
	if err != nil {
		log.Fatalf("❌ Failed to connect RabbitMQ: %v", err)
	}
	defer rabbit.Close()

	// Ensure queue exists
	_, err = rabbit.DeclareQueue("checkout_requested")
	if err != nil {
		log.Fatalf("RabbitMQ Failed to declare queue: %v", err)
	}

	// Start consumer
	err = rabbit.Consume("checkout_requested")
	if err != nil {
		log.Fatalf("❌ Failed to start consumer: %v", err)
	}

	log.Println("✅ RabbitMQ Consumer Started")

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
