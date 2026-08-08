package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type AuthResponseDTO struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

func ToAuthResponseDTO(res *domain.AuthResponse) AuthResponseDTO {
	return AuthResponseDTO{
		Token:     res.Token,
		ExpiresAt: res.ExpiresAt,
		User:      ToUserResponse(&res.User),
	}
}
