package dto

import (
	"fmt"
	"net/url"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type UserRegisterRequest struct {
	Name       string `json:"name" validate:"required" example:"Budi Haryanto"`
	Email      string `json:"email" validate:"required,email" example:"budi@sikon.com"`
	Password   string `json:"password" validate:"required,min=6" example:"rahasia123"`
	Role       string `json:"role" validate:"required,oneof=admin sales" example:"sales"`
	ImageURL   string `json:"image_url" validate:"omitempty" example:"https://example.com/image.jpg"`
	Phone      string `json:"phone" validate:"omitempty" example:"6281200000001"`
	StatusText string `json:"status_text" validate:"omitempty" example:"Online sekarang"`
	IsActive   *bool  `json:"is_active" validate:"omitempty"`
	SortOrder  int    `json:"sort_order" validate:"omitempty"`
}

type UserUpdateRequest struct {
	Name       string `json:"name" validate:"omitempty" example:"Budi Haryanto Update"`
	Role       string `json:"role" validate:"omitempty,oneof=admin sales" example:"admin"`
	ImageURL   string `json:"image_url" validate:"omitempty"`
	Phone      string `json:"phone" validate:"omitempty" example:"6281200000001"`
	StatusText string `json:"status_text" validate:"omitempty" example:"Online sekarang"`
	IsActive   *bool  `json:"is_active" validate:"omitempty"`
	SortOrder  *int   `json:"sort_order" validate:"omitempty"`
}

type UserResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	ImageURL   string    `json:"image_url"`
	Phone      string    `json:"phone"`
	StatusText string    `json:"status_text"`
	IsActive   bool      `json:"is_active"`
	SortOrder  int       `json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Response khusus untuk landing page / widget WhatsApp
type PublicSalesResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ImageURL    string `json:"image_url"`
	Phone       string `json:"phone"`
	StatusText  string `json:"status_text"`
	WhatsAppURL string `json:"whatsapp_url"`
}

func ToUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:         user.ID,
		Name:       user.Name,
		Email:      user.Email,
		Role:       string(user.Role),
		ImageURL:   user.ImageURL,
		Phone:      user.Phone,
		StatusText: user.StatusText,
		IsActive:   user.IsActive,
		SortOrder:  user.SortOrder,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
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

func ToPublicSalesResponse(user *domain.User) PublicSalesResponse {
	defaultMsg := "Halo, saya ingin konsultasi mengenai produk Konveksi. Mohon info lebih lanjut, terima kasih."
	encodedMsg := url.QueryEscape(defaultMsg)
	waURL := fmt.Sprintf("https://api.whatsapp.com/send/?phone=%s&text=%s&type=phone_number&app_absent=0", user.Phone, encodedMsg)

	return PublicSalesResponse{
		ID:          user.ID,
		Name:        user.Name,
		ImageURL:    user.ImageURL,
		Phone:       user.Phone,
		StatusText:  user.StatusText,
		WhatsAppURL: waURL,
	}
}

func ToPublicSalesResponseList(users []domain.User) []PublicSalesResponse {
	var responses []PublicSalesResponse
	for _, u := range users {
		responses = append(responses, ToPublicSalesResponse(&u))
	}
	if responses == nil {
		return []PublicSalesResponse{}
	}
	return responses
}
