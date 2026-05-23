package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type UserRegisterRequest struct {
	Name     string `json:"name" validate:"required" example:"Budi Haryanto"`
	Email    string `json:"email" validate:"required,email" example:"budi@sikon.com"`
	Password string `json:"password" validate:"required,min=6" example:"rahasia123"`
	Role     string `json:"role" validate:"required,oneof=admin sales" example:"sales"`
}

type UserUpdateRequest struct {
	Name string `json:"name" validate:"omitempty" example:"Budi Haryanto Update"`
	Role string `json:"role" validate:"omitempty,oneof=admin sales" example:"admin"`
}

// --- RESPONSE ---
type UserResponse struct {
	ID        string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string    `json:"name" example:"Budi Haryanto"`
	Email     string    `json:"email" example:"budi@sikon.com"`
	Role      string    `json:"role" example:"sales"`
	CreatedAt time.Time `json:"created_at" example:"2023-10-01T15:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-10-01T15:00:00Z"`
}

func ToUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func ToUserResponseList(users []domain.User) []UserResponse {
	var responses []UserResponse
	for _, u := range users {
		responses = append(responses, ToUserResponse(&u))
	}
	if responses == nil {
		return []UserResponse{}
	}
	return responses
}
