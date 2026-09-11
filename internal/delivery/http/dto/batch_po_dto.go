package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---

type BatchPOCreateRequest struct {
	Name        string    `json:"name" validate:"required" example:"PO 3 September 2026"`
	TargetMonth int       `json:"target_month" validate:"required,min=1,max=12" example:"9"`
	TargetYear  int       `json:"target_year" validate:"required,min=2000" example:"2026"`
	OpenDate    time.Time `json:"open_date" validate:"required" example:"2026-09-05T00:00:00Z"`
	CloseDate   time.Time `json:"close_date" validate:"required" example:"2026-09-12T00:00:00Z"`
	StartDate   time.Time `json:"start_date" validate:"required" example:"2026-09-12T00:00:00Z"`
	EndDate     time.Time `json:"end_date" validate:"required" example:"2026-09-19T00:00:00Z"`
	Quota       int       `json:"quota" validate:"gte=0" example:"100"` // 0 berarti unlimited
}

type BatchPOUpdateRequest struct {
	Name        string     `json:"name" validate:"omitempty" example:"PO 3 September 2026 Revisi"`
	TargetMonth *int       `json:"target_month,omitempty" validate:"omitempty,min=1,max=12" example:"9"`
	TargetYear  *int       `json:"target_year,omitempty" validate:"omitempty,min=2000" example:"2026"`
	OpenDate    *time.Time `json:"open_date" validate:"omitempty" example:"2026-09-05T00:00:00Z"`
	CloseDate   *time.Time `json:"close_date" validate:"omitempty" example:"2026-09-12T00:00:00Z"`
	StartDate   *time.Time `json:"start_date" validate:"omitempty" example:"2026-09-12T00:00:00Z"`
	EndDate     *time.Time `json:"end_date" validate:"omitempty" example:"2026-09-19T00:00:00Z"`
	Quota       *int       `json:"quota" validate:"omitempty,gte=0"`
}

type BatchPOStatusUpdateRequest struct {
	Status string `json:"status" validate:"required,oneof=draft active closed" example:"active"`
}

// --- RESPONSE ---

type BatchPOResponse struct {
	ID          string    `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string    `json:"name" example:"PO 3 September 2026"`
	TargetMonth int       `json:"target_month" example:"9"`
	TargetYear  int       `json:"target_year" example:"2026"`
	OpenDate    time.Time `json:"open_date" example:"2026-09-05T00:00:00Z"`
	CloseDate   time.Time `json:"close_date" example:"2026-09-12T00:00:00Z"`
	StartDate   time.Time `json:"start_date" example:"2026-09-12T00:00:00Z"`
	EndDate     time.Time `json:"end_date" example:"2026-09-19T00:00:00Z"`
	Status      string    `json:"status" example:"active"`
	Quota       int       `json:"quota" example:"100"`
	CreatedAt   time.Time `json:"created_at" example:"2026-09-05T10:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2026-09-05T10:00:00Z"`
}

func ToBatchPOResponse(b *domain.BatchPO) BatchPOResponse {
	return BatchPOResponse{
		ID:          b.ID,
		Name:        b.Name,
		TargetMonth: b.TargetMonth,
		TargetYear:  b.TargetYear,
		OpenDate:    b.OpenDate,
		CloseDate:   b.CloseDate,
		StartDate:   b.StartDate,
		EndDate:     b.EndDate,
		Status:      string(b.Status),
		Quota:       b.Quota,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

func ToBatchPOResponseList(list []domain.BatchPO) []BatchPOResponse {
	if len(list) == 0 {
		return []BatchPOResponse{}
	}
	responses := make([]BatchPOResponse, len(list))
	for i, b := range list {
		responses[i] = ToBatchPOResponse(&b)
	}
	return responses
}

// BatchPOSuggestedOpenDateResponse — dipakai FE untuk auto-prefill form "Buat PO Baru".
// open_date bernilai null kalau belum ada PO sama sekali (Admin isi manual untuk PO pertama).
type BatchPOSuggestedOpenDateResponse struct {
	OpenDate *time.Time `json:"open_date"`
}
