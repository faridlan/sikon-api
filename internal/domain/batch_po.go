package domain

import (
	"context"
	"time"
)

type BatchPOStatus string

const (
	BatchPOStatusDraft  BatchPOStatus = "draft"
	BatchPOStatusActive BatchPOStatus = "active"
	BatchPOStatusClosed BatchPOStatus = "closed"
)

type BatchPO struct {
	ID          string
	Name        string
	TargetMonth int
	TargetYear  int
	StartDate   time.Time
	EndDate     time.Time
	Status      BatchPOStatus
	Quota       int // 0 berarti tidak terbatas (unlimited)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type BatchPOCreateInput struct {
	Name        string
	TargetMonth int
	TargetYear  int
	StartDate   time.Time
	EndDate     time.Time
	Quota       int
}

type BatchPOUpdateInput struct {
	Name        string
	TargetMonth *int
	TargetYear  *int
	StartDate   *time.Time
	EndDate     *time.Time
	Status      BatchPOStatus
	Quota       *int
}

type BatchPORepository interface {
	Create(ctx context.Context, batchPO *BatchPO) error
	GetByID(ctx context.Context, id string) (*BatchPO, error)
	Fetch(ctx context.Context, limit, offset int) ([]BatchPO, int64, error)
	FetchActive(ctx context.Context) ([]BatchPO, error) // Khusus untuk dropdown Sales
	Update(ctx context.Context, batchPO *BatchPO) error
	Delete(ctx context.Context, id string) error
	GetActivePOByDate(ctx context.Context, targetDate time.Time) (*BatchPO, error)
}

type BatchPOUsecase interface {
	CreateBatchPO(ctx context.Context, input BatchPOCreateInput) (*BatchPO, error)
	GetBatchPO(ctx context.Context, id string) (*BatchPO, error)
	ListBatchPOs(ctx context.Context, query PaginationQuery) ([]BatchPO, PaginationMeta, error)
	ListActiveBatchPOs(ctx context.Context) ([]BatchPO, error)
	UpdateBatchPO(ctx context.Context, id string, input BatchPOUpdateInput) (*BatchPO, error)
	DeleteBatchPO(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status BatchPOStatus) error
}

func (s BatchPOStatus) IsValid() bool {
	switch s {
	case BatchPOStatusDraft, BatchPOStatusActive, BatchPOStatusClosed:
		return true
	}
	return false
}
