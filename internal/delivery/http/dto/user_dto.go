package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type UserRegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required,oneof=admin sales"` // oneof memvalidasi enum
}

type UserUpdateRequest struct {
	Name string `json:"name" validate:"omitempty"`
	Role string `json:"role" validate:"omitempty,oneof=admin sales"`
}

// --- RESPONSE ---
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Mapper: Mengubah domain.User menjadi UserResponse
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
	// Pastikan mengembalikan slice kosong [] bukan null jika tidak ada data
	if responses == nil {
		return []UserResponse{}
	}
	return responses
}
