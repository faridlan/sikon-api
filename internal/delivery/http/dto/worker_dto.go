package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type WorkerCreateRequest struct {
	UserID     *string `json:"user_id" validate:"omitempty,uuid" example:"110e8400-e29b-41d4-a716-446655440000"`
	Name       string  `json:"name" validate:"required" example:"Mang Ade"`
	Phone      string  `json:"phone" validate:"omitempty" example:"081234567890"`
	Role       string  `json:"role" validate:"required,oneof=tailor cutter finishing sales staff helper" example:"tailor"`
	SalaryType string  `json:"salary_type" validate:"required,oneof=piece_rate daily monthly" example:"piece_rate"`
	DailyRate  float64 `json:"daily_rate" validate:"gte=0" example:"66666.67"`
}

type WorkerUpdateRequest struct {
	UserID     *string `json:"user_id" validate:"omitempty,uuid" example:"110e8400-e29b-41d4-a716-446655440000"`
	Name       string  `json:"name" validate:"omitempty" example:"Mang Ade Supriatna"`
	Phone      string  `json:"phone" validate:"omitempty" example:"081234567890"`
	Role       string  `json:"role" validate:"omitempty,oneof=tailor cutter finishing sales staff helper" example:"tailor"`
	SalaryType string  `json:"salary_type" validate:"omitempty,oneof=piece_rate daily monthly" example:"piece_rate"`
	DailyRate  float64 `json:"daily_rate" validate:"gte=0" example:"66666.67"`
	Status     string  `json:"status" validate:"omitempty,oneof=active inactive" example:"active"`
}

// --- RESPONSE ---
type WorkerResponse struct {
	ID         string        `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID     *string       `json:"user_id,omitempty" example:"110e8400-e29b-41d4-a716-446655440000"`
	Name       string        `json:"name" example:"Mang Ade"`
	Phone      string        `json:"phone" example:"081234567890"`
	Role       string        `json:"role" example:"tailor"`
	SalaryType string        `json:"salary_type" example:"piece_rate"`
	DailyRate  float64       `json:"daily_rate" example:"66666.67"`
	Status     string        `json:"status" example:"active"`
	CreatedAt  time.Time     `json:"created_at" example:"2026-08-20T15:00:00Z"`
	UpdatedAt  time.Time     `json:"updated_at" example:"2026-08-20T15:00:00Z"`
	User       *UserResponse `json:"user,omitempty"`
}

func ToWorkerResponse(worker *domain.Worker) WorkerResponse {
	if worker == nil {
		return WorkerResponse{}
	}

	resp := WorkerResponse{
		ID:         worker.ID,
		UserID:     worker.UserID,
		Name:       worker.Name,
		Phone:      worker.Phone,
		Role:       string(worker.Role),
		SalaryType: string(worker.SalaryType),
		DailyRate:  worker.DailyRate,
		Status:     string(worker.Status),
		CreatedAt:  worker.CreatedAt,
		UpdatedAt:  worker.UpdatedAt,
	}

	if worker.User != nil {
		userResp := ToUserResponse(worker.User)
		resp.User = &userResp
	}

	return resp
}

func ToWorkerResponseList(workers []domain.Worker) []WorkerResponse {
	var responses []WorkerResponse
	for _, w := range workers {
		responses = append(responses, ToWorkerResponse(&w))
	}
	if responses == nil {
		return []WorkerResponse{}
	}
	return responses
}
