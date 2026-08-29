package models

import "time"

// Status pesanan 
const (
	StatusPending   = "pending"
	StatusCooking   = "cooking"
	StatusDone      = "done"
	StatusCancelled = "cancelled"
)

// Detail Item
type OrderItem struct {
	ID       int64  `json:"id"`
	OrderID  int64  `json:"order_id"`
	MenuName string `json:"menu_name"`
	Quantity int    `json:"quantity"`
	Note     string `json:"note"`
}

// Pesanan
type Order struct {
	ID           int64       `json:"id"`
	QueueNumber  int         `json:"queue_number"`
	CustomerName string      `json:"customer_name"`
	Status       string      `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Items        []OrderItem `json:"items"`
}