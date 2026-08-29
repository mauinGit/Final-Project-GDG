package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Demi Keamanan Maul:D
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}