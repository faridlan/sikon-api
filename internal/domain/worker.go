package domain

import (
	"context"
	"time"
)

type WorkerRole string
type WorkerSalaryType string
type WorkerStatus string

const (
	WorkerRoleTailor    WorkerRole = "tailor"
	WorkerRoleCutter    WorkerRole = "cutter"
	WorkerRoleFinishing WorkerRole = "finishing"
	WorkerRoleHelper    WorkerRole = "helper"

	WorkerSalaryTypePieceRate WorkerSalaryType = "piece_rate" // Borongan per pcs
	WorkerSalaryTypeDaily     WorkerSalaryType = "daily"      // Harian
	WorkerSalaryTypeMonthly   WorkerSalaryType = "monthly"    // Bulanan

	WorkerStatusActive   WorkerStatus = "active"
	WorkerStatusInactive WorkerStatus = "inactive"
)

type Worker struct {
	ID         string
	Name       string
	Phone      string
	Role       WorkerRole
	SalaryType WorkerSalaryType
	Status     WorkerStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Input Struct
type WorkerCreateInput struct {
	Name       string
	Phone      string
	Role       WorkerRole
	SalaryType WorkerSalaryType
}

type WorkerUpdateInput struct {
	Name       string
	Phone      string
	Role       WorkerRole
	SalaryType WorkerSalaryType
	Status     WorkerStatus
}

type WorkerFilter struct {
	Search     string           // Pencarian Nama atau Nomor Telepon
	Role       WorkerRole       // Filter spesifik Peran (tailor/cutter/dll)
	SalaryType WorkerSalaryType // Filter Tipe Gaji (piece_rate/daily/monthly)
	Status     WorkerStatus     // Filter Status Pekerja
}

type WorkerRepository interface {
	Create(ctx context.Context, worker *Worker) error
	GetByID(ctx context.Context, id string) (*Worker, error)
	Fetch(ctx context.Context, filter WorkerFilter, limit, offset int) ([]Worker, int64, error)
	Update(ctx context.Context, worker *Worker) error
	Delete(ctx context.Context, id string) error
}

type WorkerUsecase interface {
	CreateWorker(ctx context.Context, input WorkerCreateInput) (*Worker, error)
	GetWorker(ctx context.Context, id string) (*Worker, error)
	ListWorkers(ctx context.Context, query PaginationQuery, filter WorkerFilter) ([]Worker, PaginationMeta, error)
	UpdateWorker(ctx context.Context, id string, input WorkerUpdateInput) (*Worker, error)
	DeleteWorker(ctx context.Context, id string) error
}
