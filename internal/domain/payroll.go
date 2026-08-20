package domain

import (
	"context"
	"time"
)

type PayrollStatus string

const (
	PayrollStatusDraft    PayrollStatus = "draft"
	PayrollStatusApproved PayrollStatus = "approved"
	PayrollStatusPaid     PayrollStatus = "paid"
)

type Payroll struct {
	ID            string
	PayrollNumber string
	StartDate     time.Time
	EndDate       time.Time
	TotalAmount   float64
	Status        PayrollStatus
	ExpenseID     *string
	CreatedByID   string
	PaidAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Creator  *User
	Expense  *Expense
	WorkLogs []WorkLog
}

// Input Struct
type PayrollCreateInput struct {
	StartDate   time.Time
	EndDate     time.Time
	WorkLogIDs  []string // ID work_logs yang akan digabungkan ke rekap gaji ini
	CreatedByID string
}

type PayrollUpdateStatusInput struct {
	Status     PayrollStatus
	OperatorID string
}

type PayrollFilter struct {
	Status    PayrollStatus
	StartDate *time.Time
	EndDate   *time.Time
}

type PayrollRepository interface {
	Create(ctx context.Context, payroll *Payroll, workLogIDs []string) error
	GetByID(ctx context.Context, id string) (*Payroll, error)
	Fetch(ctx context.Context, filter PayrollFilter, limit, offset int) ([]Payroll, int64, error)
	UpdateStatus(ctx context.Context, id string, status PayrollStatus, expenseID *string, paidAt *time.Time) error
	Delete(ctx context.Context, id string) error
}

type PayrollUsecase interface {
	CreatePayroll(ctx context.Context, input PayrollCreateInput) (*Payroll, error)
	GetPayroll(ctx context.Context, id string) (*Payroll, error)
	ListPayrolls(ctx context.Context, query PaginationQuery, filter PayrollFilter) ([]Payroll, PaginationMeta, error)
	ProcessPayrollPayment(ctx context.Context, id string, operatorID string) (*Payroll, error)
	DeletePayroll(ctx context.Context, id string) error
}
