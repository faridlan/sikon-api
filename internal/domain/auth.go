package domain

import (
	"context"
	"time"
)

// AuthClaims menyimpan payload data user di dalam JWT Token
type AuthClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   Role   `json:"role"`
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

type AuthUsecase interface {
	Login(ctx context.Context, input LoginInput) (*AuthResponse, error)
}
