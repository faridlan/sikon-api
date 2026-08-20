package domain

import (
	"context"
	"time"
)

type JobType string

const (
	JobTypeJahit     JobType = "jahit"
	JobTypePotong    JobType = "potong"
	JobTypeBordir    JobType = "bordir"
	JobTypeFinishing JobType = "finishing"
)

type WorkLog struct {
	ID          string
	WorkerID    string
	BatchPoID   *string
	PayrollID   *string
	JobType     JobType
	Qty         int
	RatePerQty  float64
	TotalAmount float64
	WorkDate    time.Time
	Notes       string
	CreatedByID string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Worker  *Worker
	BatchPO *BatchPO
	Creator *User
}

// Input Struct
type WorkLogCreateInput struct {
	WorkerID    string
	BatchPoID   *string
	JobType     JobType
	Qty         int
	RatePerQty  float64
	WorkDate    time.Time
	Notes       string
	CreatedByID string
}

type WorkLogUpdateInput struct {
	WorkerID   string
	BatchPoID  *string
	JobType    JobType
	Qty        int
	RatePerQty float64
	WorkDate   time.Time
	Notes      string
}

type WorkLogFilter struct {
	WorkerID  *string
	BatchPoID *string
	PayrollID *string
	JobType   JobType
	StartDate *time.Time
	EndDate   *time.Time
	IsUnpaid  bool // true: filter work_log yang payroll_id NULL (belum digaji)
}

type WorkLogRepository interface {
	Create(ctx context.Context, log *WorkLog) error
	GetByID(ctx context.Context, id string) (*WorkLog, error)
	Fetch(ctx context.Context, filter WorkLogFilter, limit, offset int) ([]WorkLog, int64, error)
	Update(ctx context.Context, log *WorkLog) error
	Delete(ctx context.Context, id string) error
}

type WorkLogUsecase interface {
	CreateWorkLog(ctx context.Context, input WorkLogCreateInput) (*WorkLog, error)
	GetWorkLog(ctx context.Context, id string) (*WorkLog, error)
	ListWorkLogs(ctx context.Context, query PaginationQuery, filter WorkLogFilter) ([]WorkLog, PaginationMeta, error)
	UpdateWorkLog(ctx context.Context, id string, input WorkLogUpdateInput) (*WorkLog, error)
	DeleteWorkLog(ctx context.Context, id string) error
}
