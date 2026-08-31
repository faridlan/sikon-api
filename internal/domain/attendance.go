package domain

import (
	"context"
	"time"
)

type AttendanceStatus string

const (
	AttendanceStatusPresent    AttendanceStatus = "present"    // Hadir (WorkDurationIndex = 1.00)
	AttendanceStatusHalfDay    AttendanceStatus = "half_day"   // Setengah Hari (WorkDurationIndex = 0.50)
	AttendanceStatusPermission AttendanceStatus = "permission" // Izin / Sakit (WorkDurationIndex = 0.00)
	AttendanceStatusAlpha      AttendanceStatus = "alpha"      // Tanpa Keterangan (WorkDurationIndex = 0.00)
)

type Attendance struct {
	ID                string
	WorkerID          string
	PayrollID         *string
	AttendanceDate    time.Time
	Status            AttendanceStatus
	WorkDurationIndex float64 // 1.00, 0.50, atau 0.00
	DailyRate         float64 // Tarif harian saat dicatat
	TotalAmount       float64 // WorkDurationIndex * DailyRate
	Notes             string
	CreatedByID       string
	CreatedAt         time.Time
	UpdatedAt         time.Time

	Worker  *Worker  // Relasi ke Worker
	Payroll *Payroll // Relasi ke Payroll (jika sudah dicairkan)
	Creator *User    // Admin/Personalia pencatat
}

// Input Struct Single Record
type AttendanceCreateInput struct {
	WorkerID          string           `json:"worker_id"`
	AttendanceDate    time.Time        `json:"attendance_date"`
	Status            AttendanceStatus `json:"status"`
	WorkDurationIndex float64          `json:"work_duration_index"` // Opsional, jika diisi 0 maka dihitung otomatis berdasar Status
	Notes             string           `json:"notes"`
	CreatedByID       string           `json:"-"`
}

// Input Struct Batch Record (Untuk mencatat absensi banyak karyawan sekaligus dalam 1 hari)
type BatchAttendanceItem struct {
	WorkerID          string           `json:"worker_id"`
	Status            AttendanceStatus `json:"status"`
	WorkDurationIndex *float64         `json:"work_duration_index"` // Opsional override
	Notes             string           `json:"notes"`
}

type BatchAttendanceInput struct {
	AttendanceDate time.Time             `json:"attendance_date"`
	Items          []BatchAttendanceItem `json:"items"`
	CreatedByID    string                `json:"-"`
}

type AttendanceUpdateInput struct {
	Status            AttendanceStatus `json:"status"`
	WorkDurationIndex *float64         `json:"work_duration_index"`
	Notes             string           `json:"notes"`
}

type AttendanceFilter struct {
	WorkerID  *string
	PayrollID *string
	Status    AttendanceStatus
	StartDate *time.Time
	EndDate   *time.Time
	IsUnpaid  bool // true: filter attendance yang payroll_id NULL
}

type AttendanceRepository interface {
	Create(ctx context.Context, att *Attendance) error
	CreateBatch(ctx context.Context, attendances []Attendance) error
	GetByID(ctx context.Context, id string) (*Attendance, error)
	GetByWorkerAndDate(ctx context.Context, workerID string, date time.Time) (*Attendance, error)
	Fetch(ctx context.Context, filter AttendanceFilter, limit, offset int) ([]Attendance, int64, error)
	Update(ctx context.Context, att *Attendance) error
	Delete(ctx context.Context, id string) error

	// Helper untuk Rekap Payroll Harian
	GetTotalAmountByWorkerAndPeriod(ctx context.Context, workerID string, startDate, endDate time.Time) (float64, int, error)
}

type AttendanceUsecase interface {
	RecordAttendance(ctx context.Context, input AttendanceCreateInput) (*Attendance, error)
	RecordBatchAttendance(ctx context.Context, input BatchAttendanceInput) ([]Attendance, error) // 👈 Absensi Masal Harian
	GetAttendance(ctx context.Context, id string) (*Attendance, error)
	ListAttendances(ctx context.Context, query PaginationQuery, filter AttendanceFilter) ([]Attendance, PaginationMeta, error)
	UpdateAttendance(ctx context.Context, id string, input AttendanceUpdateInput) (*Attendance, error)
	DeleteAttendance(ctx context.Context, id string) error
}
