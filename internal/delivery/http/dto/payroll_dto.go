package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type PayrollCreateRequest struct {
	StartDate     string   `json:"start_date" validate:"required" example:"2026-08-14"`
	EndDate       string   `json:"end_date" validate:"required" example:"2026-08-20"`
	WorkLogIDs    []string `json:"work_log_ids" validate:"omitempty,dive,uuid" example:"[\"770e8400-e29b-41d4-a716-446655440000\"]"`
	AttendanceIDs []string `json:"attendance_ids" validate:"omitempty,dive,uuid" example:"[\"880e8400-e29b-41d4-a716-446655440000\"]"`
}

// --- RESPONSE ---
type PayrollResponse struct {
	ID            string     `json:"id" example:"880e8400-e29b-41d4-a716-446655440000"`
	PayrollNumber string     `json:"payroll_number" example:"PAY-202608-001"`
	StartDate     string     `json:"start_date" example:"2026-08-14"`
	EndDate       string     `json:"end_date" example:"2026-08-20"`
	TotalAmount   float64    `json:"total_amount" example:"1500000"`
	Status        string     `json:"status" example:"draft"`
	ExpenseID     *string    `json:"expense_id,omitempty" example:"990e8400-e29b-41d4-a716-446655440000"`
	CreatedByID   string     `json:"created_by" example:"110e8400-e29b-41d4-a716-446655440000"`
	PaidAt        *time.Time `json:"paid_at,omitempty" example:"2026-08-20T15:00:00Z"`
	CreatedAt     time.Time  `json:"created_at" example:"2026-08-20T15:00:00Z"`
	UpdatedAt     time.Time  `json:"updated_at" example:"2026-08-20T15:00:00Z"`

	Creator     *UserResponse        `json:"creator,omitempty"`
	Expense     *ExpenseResponse     `json:"expense,omitempty"`
	WorkLogs    []WorkLogResponse    `json:"work_logs,omitempty"`
	Attendances []AttendanceResponse `json:"attendances,omitempty"` // 👈 Tambahkan array Attendance Response
}

func ToPayrollResponse(payroll *domain.Payroll) PayrollResponse {
	if payroll == nil {
		return PayrollResponse{}
	}

	resp := PayrollResponse{
		ID:            payroll.ID,
		PayrollNumber: payroll.PayrollNumber,
		StartDate:     payroll.StartDate.Format("2006-01-02"),
		EndDate:       payroll.EndDate.Format("2006-01-02"),
		TotalAmount:   payroll.TotalAmount,
		Status:        string(payroll.Status),
		ExpenseID:     payroll.ExpenseID,
		CreatedByID:   payroll.CreatedByID,
		PaidAt:        payroll.PaidAt,
		CreatedAt:     payroll.CreatedAt,
		UpdatedAt:     payroll.UpdatedAt,
	}

	if payroll.Creator != nil {
		creatorResp := ToUserResponse(payroll.Creator)
		resp.Creator = &creatorResp
	}

	if payroll.Expense != nil {
		expResp := ToExpenseResponse(payroll.Expense)
		resp.Expense = &expResp
	}

	if len(payroll.WorkLogs) > 0 {
		resp.WorkLogs = ToWorkLogResponseList(payroll.WorkLogs)
	}

	if len(payroll.Attendances) > 0 {
		resp.Attendances = ToAttendanceResponseList(payroll.Attendances)
	}

	return resp
}

func ToPayrollResponseList(payrolls []domain.Payroll) []PayrollResponse {
	var responses []PayrollResponse
	for _, p := range payrolls {
		responses = append(responses, ToPayrollResponse(&p))
	}
	if responses == nil {
		return []PayrollResponse{}
	}
	return responses
}
