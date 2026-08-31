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
	WorkerRoleSales     WorkerRole = "sales" // 👈 Tambah role sales
	WorkerRoleStaff     WorkerRole = "staff" // 👈 Tambah role staff
	WorkerRoleHelper    WorkerRole = "helper"

	WorkerSalaryTypePieceRate WorkerSalaryType = "piece_rate" // Borongan per pcs
	WorkerSalaryTypeDaily     WorkerSalaryType = "daily"      // Harian
	WorkerSalaryTypeMonthly   WorkerSalaryType = "monthly"    // Bulanan

	WorkerStatusActive   WorkerStatus = "active"
	WorkerStatusInactive WorkerStatus = "inactive"
)

type Worker struct {
	ID         string
	UserID     *string // 👈 FK ke User (Nullable, untuk staff/sales yang punya akun login)
	Name       string
	Phone      string
	Role       WorkerRole
	SalaryType WorkerSalaryType
	DailyRate  float64 // 👈 Tarif/Index harian
	Status     WorkerStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time

	User *User // 👈 Relasi ke User
}

// Input Struct
type WorkerCreateInput struct {
	UserID     *string          `json:"user_id"`
	Name       string           `json:"name"`
	Phone      string           `json:"phone"`
	Role       WorkerRole       `json:"role"`
	SalaryType WorkerSalaryType `json:"salary_type"`
	DailyRate  float64          `json:"daily_rate"` // 👈 Tambahkan DailyRate
}

type WorkerUpdateInput struct {
	UserID     *string          `json:"user_id"`
	Name       string           `json:"name"`
	Phone      string           `json:"phone"`
	Role       WorkerRole       `json:"role"`
	SalaryType WorkerSalaryType `json:"salary_type"`
	DailyRate  float64          `json:"daily_rate"` // 👈 Tambahkan DailyRate
	Status     WorkerStatus     `json:"status"`
}

type WorkerFilter struct {
	Search     string
	Role       WorkerRole
	SalaryType WorkerSalaryType
	Status     WorkerStatus
	UserID     *string // 👈 Filter by UserID jika diperlukan
}

type WorkerRepository interface {
	Create(ctx context.Context, worker *Worker) error
	GetByID(ctx context.Context, id string) (*Worker, error)
	GetByUserID(ctx context.Context, userID string) (*Worker, error) // 👈 Ambil worker berdasarkan UserID login
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
