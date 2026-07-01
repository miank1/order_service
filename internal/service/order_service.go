package service

import (
	"bytes"
	"io"
	"order_service/internal/models"
	"order_service/internal/repository"

	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type OrderService struct {
	repo              *repository.OrderRepository
	paymentServiceURL string
	httpClient        *http.Client
}

type PaymentRequest struct {
	OrderID string  `json:"order_id"`
	UserID  string  `json:"user_id"`
	Amount  float64 `json:"amount"`
}

func NewOrderService(repo *repository.OrderRepository, paymentURL string) *OrderService {
	return &OrderService{
		repo:              repo,
		paymentServiceURL: paymentURL,
		httpClient:        &http.Client{},
	}
}

type OrderItemReq struct {
	ProductID   string  `json:"product_id"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	ProductName string  `json:"product_name"`
	Category    string  `json:"category"`
}

func (s *OrderService) CreateOrder(userID string, items []OrderItemReq) (*models.Order, error) {

	var total float64

	for _, i := range items {
		total += float64(i.Quantity) * i.Price
	}

	order := &models.Order{
		UserID:     userID,
		Status:     "pending",
		TotalPrice: total,
	}

	for _, i := range items {
		order.Items = append(order.Items, models.OrderItem{
			ProductID:   i.ProductID,
			Quantity:    i.Quantity,
			Price:       i.Price,
			ProductName: i.ProductName,
			Category:    i.Category,
		})
	}

	if err := s.repo.Create(order); err != nil {
		return nil, err
	}

	fmt.Println("order details", order)

	if err := s.createPayment(order); err != nil {
		return nil, err
	}

	return order, nil
}
func (s *OrderService) GetOrderByID(id string) (*models.Order, error) {
	return s.repo.GetByID(id)
}

func (s *OrderService) UpdateStatus(orderID, status string) (*models.Order, error) {
	order, err := s.repo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %v", err)
	}

	order.Status = status

	if err := s.repo.Save(order); err != nil {
		return nil, fmt.Errorf("failed to update order status: %v", err)
	} else {
		log.Println("Order status updated.")
	}

	return order, nil
}
func (s *OrderService) UpdateInventory(orderID string) error {
	order, err := s.repo.GetByID(orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("order not found")
	}

	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productServiceURL == "" {
		return fmt.Errorf("PRODUCT_SERVICE_URL not configured")
	}

	for _, item := range order.Items {

		// Correct endpoint with product_id in URL
		url := fmt.Sprintf("%s/api/v1/products/%s/reduce-stock",
			productServiceURL, item.ProductID)

		payload := map[string]interface{}{
			"quantity": item.Quantity,
		}

		fmt.Println("Payload is ************* ", payload)

		body, _ := json.Marshal(payload)

		req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
		if err != nil {
			fmt.Printf("❌ Failed creating request: %v\n", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)

		log.Println("response is --------------- ", resp)
		if err != nil {
			fmt.Printf("❌ Failed to reduce stock for %s: %v\n", item.ProductID, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("⚠️ Stock update failed for product %s (status %d)\n",
				item.ProductID, resp.StatusCode)
		} else {
			fmt.Printf("✅ Stock updated for product %s\n", item.ProductID)
		}

		resp.Body.Close()
	}

	return nil
}

func (s *OrderService) createPayment(order *models.Order) error {

	payload := PaymentRequest{
		OrderID: order.ID,
		UserID:  order.UserID,
		Amount:  order.TotalPrice,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	log.Println(
		"Calling Payment Service:",
		s.paymentServiceURL+"/api/v1/internal/payments",
	)

	resp, err := s.httpClient.Post(
		s.paymentServiceURL+"/api/v1/internal/payments",
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)

		return fmt.Errorf(
			"payment service failed: %s",
			string(respBody),
		)
	}

	return nil
}
