package models

import "time"

type MenuItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Price     int       `json:"price"`
	Category  string    `json:"category"`
	ImageURL  *string   `json:"image_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}