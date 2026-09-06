package models

import "time"

// Status pesanan 
const (
	StatusPending   = "pending"
	StatusCooking   = "cooking"
	StatusDone      = "done"
	StatusCancelled = "cancelled"
	PaymentCash    = "cash"
	PaymentNonCash = "non_cash"
)

// Detail Item
type OrderItem struct {
	ID           int64  		`json:"id"`
	OrderID      int64  		`json:"order_id"`
	MenuName     string 		`json:"menu_name"`
	Quantity     int    		`json:"quantity"`
	Note         string 		`json:"note"`
	MenuItemID   int64 		`json:"menu_item_id"`
	PriceAtOrder int   		`json:"price_at_order"`
}

// Pesanan
type Order struct {
	ID           int64       	`json:"id"`
	QueueNumber  int         	`json:"queue_number"`
	CustomerName string      	`json:"customer_name"`
	Status       string      	`json:"status"`
	Subtotal      int    		`json:"subtotal"`
	Discount      int    		`json:"discount"`
	Total         int    		`json:"total"`
	PaymentMethod string 		`json:"payment_method"`
	AmountPaid    int    		`json:"amount_paid"`
	ChangeAmount  int    		`json:"change_amount"`
	CreatedAt    time.Time   	`json:"created_at"`
	UpdatedAt    time.Time   	`json:"updated_at"`
	Items        []OrderItem 	`json:"items"`
}