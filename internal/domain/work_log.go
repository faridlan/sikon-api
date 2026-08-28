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
	OrderID     *string // 👈 FK ke Order Konsumen Spesifik
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
	Order   *Order // 👈 Relasi ke Konsumen
	Creator *User
}

// Input Struct
type WorkLogCreateInput struct {
	WorkerID    string
	BatchPoID   *string
	OrderID     *string // 👈 Tambahkan OrderID
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
	OrderID    *string // 👈 Tambahkan OrderID
	JobType    JobType
	Qty        int
	RatePerQty float64
	WorkDate   time.Time
	Notes      string
}

type DistributeWorkLoadInput struct {
	BatchPOID   string
	JobType     JobType
	WorkerIDs   []string
	RatePerQty  float64
	WorkDate    time.Time
	Notes       string
	CreatedByID string
}

type WorkLogFilter struct {
	WorkerID  *string
	BatchPoID *string
	OrderID   *string // 👈 Filter per Order/Konsumen
	PayrollID *string
	JobType   JobType
	StartDate *time.Time
	EndDate   *time.Time
	IsUnpaid  bool
}

type WorkLogRepository interface {
	Create(ctx context.Context, log *WorkLog) error
	CreateBatch(ctx context.Context, logs []WorkLog) error
	GetByID(ctx context.Context, id string) (*WorkLog, error)
	Fetch(ctx context.Context, filter WorkLogFilter, limit, offset int) ([]WorkLog, int64, error)
	Update(ctx context.Context, log *WorkLog) error
	Delete(ctx context.Context, id string) error

	// Helper untuk Guard Gembok Qty & HPP
	GetTotalQtyByOrderAndJobType(ctx context.Context, orderID string, jobType JobType, excludeLogID string) (int, error)
	GetTotalCostByOrder(ctx context.Context, orderID string) (float64, error)
	GetTotalCostByBatchPO(ctx context.Context, batchPoID string) (float64, error)
}

type WorkLogUsecase interface {
	CreateWorkLog(ctx context.Context, input WorkLogCreateInput) (*WorkLog, error)
	DistributeWorkLoad(ctx context.Context, input DistributeWorkLoadInput) ([]WorkLog, error)
	GetWorkLog(ctx context.Context, id string) (*WorkLog, error)
	ListWorkLogs(ctx context.Context, query PaginationQuery, filter WorkLogFilter) ([]WorkLog, PaginationMeta, error)
	UpdateWorkLog(ctx context.Context, id string, input WorkLogUpdateInput) (*WorkLog, error)
	DeleteWorkLog(ctx context.Context, id string) error
}
