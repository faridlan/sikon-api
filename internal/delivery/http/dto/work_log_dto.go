package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type WorkLogCreateRequest struct {
	WorkerID   string  `json:"worker_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	BatchPoID  *string `json:"batch_po_id" validate:"omitempty,uuid" example:"660e8400-e29b-41d4-a716-446655440000"`
	OrderID    *string `json:"order_id" validate:"omitempty,uuid" example:"990e8400-e29b-41d4-a716-446655440000"` // 👈 Tambah OrderID
	JobType    string  `json:"job_type" validate:"required,oneof=jahit potong bordir finishing" example:"jahit"`
	Qty        int     `json:"qty" validate:"required,gt=0" example:"5"`
	RatePerQty float64 `json:"rate_per_qty" validate:"required,gt=0" example:"12000"`
	WorkDate   string  `json:"work_date" validate:"required" example:"2026-08-20"`
	Notes      string  `json:"notes" validate:"omitempty" example:"Tambahan resleting dada"`
}

type WorkLogUpdateRequest struct {
	WorkerID   string  `json:"worker_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	BatchPoID  *string `json:"batch_po_id" validate:"omitempty,uuid" example:"660e8400-e29b-41d4-a716-446655440000"`
	OrderID    *string `json:"order_id" validate:"omitempty,uuid" example:"990e8400-e29b-41d4-a716-446655440000"` // 👈 Tambah OrderID
	JobType    string  `json:"job_type" validate:"omitempty,oneof=jahit potong bordir finishing" example:"jahit"`
	Qty        int     `json:"qty" validate:"omitempty,gt=0" example:"5"`
	RatePerQty float64 `json:"rate_per_qty" validate:"omitempty,gt=0" example:"12000"`
	WorkDate   string  `json:"work_date" validate:"omitempty" example:"2026-08-20"`
	Notes      string  `json:"notes" validate:"omitempty" example:"Tambahan resleting dada"`
}

// 🚨 REQUEST DTO FOR AUTO DISTRIBUTE WORKLOAD
type WorkLogDistributeRequest struct {
	BatchPOID  string   `json:"batch_po_id" validate:"required,uuid" example:"660e8400-e29b-41d4-a716-446655440000"`
	JobType    string   `json:"job_type" validate:"required,oneof=jahit potong bordir finishing" example:"jahit"`
	WorkerIDs  []string `json:"worker_ids" validate:"required,gt=0,dive,uuid"`
	RatePerQty float64  `json:"rate_per_qty" validate:"required,gt=0" example:"12000"`
	WorkDate   string   `json:"work_date" validate:"omitempty" example:"2026-08-20"`
	Notes      string   `json:"notes" validate:"omitempty" example:"Pembagian otomatis borongan"`
}

// --- RESPONSE ---
type WorkLogResponse struct {
	ID          string    `json:"id" example:"770e8400-e29b-41d4-a716-446655440000"`
	WorkerID    string    `json:"worker_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	BatchPoID   *string   `json:"batch_po_id,omitempty" example:"660e8400-e29b-41d4-a716-446655440000"`
	OrderID     *string   `json:"order_id,omitempty" example:"990e8400-e29b-41d4-a716-446655440000"` // 👈 Tambah OrderID
	PayrollID   *string   `json:"payroll_id,omitempty" example:"880e8400-e29b-41d4-a716-446655440000"`
	JobType     string    `json:"job_type" example:"jahit"`
	Qty         int       `json:"qty" example:"5"`
	RatePerQty  float64   `json:"rate_per_qty" example:"12000"`
	TotalAmount float64   `json:"total_amount" example:"60000"`
	WorkDate    string    `json:"work_date" example:"2026-08-20"`
	Notes       string    `json:"notes,omitempty" example:"Tambahan resleting dada"`
	CreatedByID string    `json:"created_by" example:"110e8400-e29b-41d4-a716-446655440000"`
	CreatedAt   time.Time `json:"created_at" example:"2026-08-20T15:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2026-08-20T15:00:00Z"`

	Worker  *WorkerResponse  `json:"worker,omitempty"`
	BatchPO *BatchPOResponse `json:"batch_po,omitempty"`
	Order   *OrderResponse   `json:"order,omitempty"` // 👈 Tambah Preload Order Response
	Creator *UserResponse    `json:"creator,omitempty"`
}

func ToWorkLogResponse(log *domain.WorkLog) WorkLogResponse {
	if log == nil {
		return WorkLogResponse{}
	}

	resp := WorkLogResponse{
		ID:          log.ID,
		WorkerID:    log.WorkerID,
		BatchPoID:   log.BatchPoID,
		OrderID:     log.OrderID, // 👈 Map OrderID
		PayrollID:   log.PayrollID,
		JobType:     string(log.JobType),
		Qty:         log.Qty,
		RatePerQty:  log.RatePerQty,
		TotalAmount: log.TotalAmount,
		WorkDate:    log.WorkDate.Format("2006-01-02"),
		Notes:       log.Notes,
		CreatedByID: log.CreatedByID,
		CreatedAt:   log.CreatedAt,
		UpdatedAt:   log.UpdatedAt,
	}

	if log.Worker != nil {
		workerResp := ToWorkerResponse(log.Worker)
		resp.Worker = &workerResp
	}

	if log.BatchPO != nil {
		poResp := ToBatchPOResponse(log.BatchPO)
		resp.BatchPO = &poResp
	}

	if log.Order != nil {
		orderResp := ToOrderResponse(log.Order) // Assumed mapper function exists
		resp.Order = &orderResp
	}

	if log.Creator != nil {
		creatorResp := ToUserResponse(log.Creator)
		resp.Creator = &creatorResp
	}

	return resp
}

func ToWorkLogResponseList(logs []domain.WorkLog) []WorkLogResponse {
	var responses []WorkLogResponse
	for _, l := range logs {
		responses = append(responses, ToWorkLogResponse(&l))
	}
	if responses == nil {
		return []WorkLogResponse{}
	}
	return responses
}
