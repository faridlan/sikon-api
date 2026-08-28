package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type AttendanceCreateRequest struct {
	WorkerID          string   `json:"worker_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	AttendanceDate    string   `json:"attendance_date" validate:"required" example:"2026-08-28"`
	Status            string   `json:"status" validate:"required,oneof=present half_day permission alpha" example:"present"`
	WorkDurationIndex *float64 `json:"work_duration_index" validate:"omitempty,gte=0,lte=1" example:"1.00"`
	Notes             string   `json:"notes" validate:"omitempty" example:"Hadir tepat waktu"`
}

type BatchAttendanceItemRequest struct {
	WorkerID          string   `json:"worker_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status            string   `json:"status" validate:"required,oneof=present half_day permission alpha" example:"present"`
	WorkDurationIndex *float64 `json:"work_duration_index" validate:"omitempty,gte=0,lte=1" example:"1.00"`
	Notes             string   `json:"notes" validate:"omitempty"`
}

type BatchAttendanceRequest struct {
	AttendanceDate string                       `json:"attendance_date" validate:"required" example:"2026-08-28"`
	Items          []BatchAttendanceItemRequest `json:"items" validate:"required,gt=0,dive"`
}

type AttendanceUpdateRequest struct {
	Status            string   `json:"status" validate:"omitempty,oneof=present half_day permission alpha" example:"half_day"`
	WorkDurationIndex *float64 `json:"work_duration_index" validate:"omitempty,gte=0,lte=1" example:"0.50"`
	Notes             string   `json:"notes" validate:"omitempty" example:"Izin pulang cepat"`
}

// --- RESPONSE ---
type AttendanceResponse struct {
	ID                string          `json:"id" example:"770e8400-e29b-41d4-a716-446655440000"`
	WorkerID          string          `json:"worker_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PayrollID         *string         `json:"payroll_id,omitempty" example:"880e8400-e29b-41d4-a716-446655440000"`
	AttendanceDate    string          `json:"attendance_date" example:"2026-08-28"`
	Status            string          `json:"status" example:"present"`
	WorkDurationIndex float64         `json:"work_duration_index" example:"1.00"`
	DailyRate         float64         `json:"daily_rate" example:"66666.67"`
	TotalAmount       float64         `json:"total_amount" example:"66666.67"`
	Notes             string          `json:"notes,omitempty" example:"Hadir tepat waktu"`
	CreatedByID       string          `json:"created_by" example:"110e8400-e29b-41d4-a716-446655440000"`
	CreatedAt         time.Time       `json:"created_at" example:"2026-08-28T08:00:00Z"`
	UpdatedAt         time.Time       `json:"updated_at" example:"2026-08-28T08:00:00Z"`
	Worker            *WorkerResponse `json:"worker,omitempty"`
	Creator           *UserResponse   `json:"creator,omitempty"`
}

func ToAttendanceResponse(att *domain.Attendance) AttendanceResponse {
	if att == nil {
		return AttendanceResponse{}
	}

	resp := AttendanceResponse{
		ID:                att.ID,
		WorkerID:          att.WorkerID,
		PayrollID:         att.PayrollID,
		AttendanceDate:    att.AttendanceDate.Format("2006-01-02"),
		Status:            string(att.Status),
		WorkDurationIndex: att.WorkDurationIndex,
		DailyRate:         att.DailyRate,
		TotalAmount:       att.TotalAmount,
		Notes:             att.Notes,
		CreatedByID:       att.CreatedByID,
		CreatedAt:         att.CreatedAt,
		UpdatedAt:         att.UpdatedAt,
	}

	if att.Worker != nil {
		workerResp := ToWorkerResponse(att.Worker)
		resp.Worker = &workerResp
	}

	if att.Creator != nil {
		creatorResp := ToUserResponse(att.Creator)
		resp.Creator = &creatorResp
	}

	return resp
}

func ToAttendanceResponseList(attendances []domain.Attendance) []AttendanceResponse {
	var responses []AttendanceResponse
	for _, a := range attendances {
		responses = append(responses, ToAttendanceResponse(&a))
	}
	if responses == nil {
		return []AttendanceResponse{}
	}
	return responses
}
