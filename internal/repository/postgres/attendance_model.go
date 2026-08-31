package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type AttendanceModel struct {
	ID                string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkerID          string         `gorm:"type:uuid;not null"`
	PayrollID         *string        `gorm:"type:uuid"`
	AttendanceDate    time.Time      `gorm:"type:date;not null"`
	Status            string         `gorm:"type:varchar(50);not null;default:'present'"`
	WorkDurationIndex float64        `gorm:"type:numeric(3,2);not null;default:1.00"`
	DailyRate         float64        `gorm:"type:numeric(15,2);not null;default:0"`
	TotalAmount       float64        `gorm:"type:numeric(15,2);not null;default:0"`
	Notes             string         `gorm:"type:text"`
	CreatedByID       string         `gorm:"column:created_by;type:uuid;not null"`
	CreatedAt         time.Time      `gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`

	// Relasi
	Worker  *WorkerModel  `gorm:"foreignKey:WorkerID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Payroll *PayrollModel `gorm:"foreignKey:PayrollID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Creator *UserModel    `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (AttendanceModel) TableName() string {
	return "attendances"
}

func (m *AttendanceModel) ToDomain() *domain.Attendance {
	att := &domain.Attendance{
		ID:                m.ID,
		WorkerID:          m.WorkerID,
		PayrollID:         m.PayrollID,
		AttendanceDate:    m.AttendanceDate,
		Status:            domain.AttendanceStatus(m.Status),
		WorkDurationIndex: m.WorkDurationIndex,
		DailyRate:         m.DailyRate,
		TotalAmount:       m.TotalAmount,
		Notes:             m.Notes,
		CreatedByID:       m.CreatedByID,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}

	if m.Worker != nil {
		att.Worker = m.Worker.ToDomain()
	}

	if m.Payroll != nil {
		att.Payroll = m.Payroll.ToDomain()
	}

	if m.Creator != nil {
		att.Creator = m.Creator.ToDomain()
	}

	return att
}

func FromAttendanceDomain(d *domain.Attendance) *AttendanceModel {
	return &AttendanceModel{
		ID:                d.ID,
		WorkerID:          d.WorkerID,
		PayrollID:         d.PayrollID,
		AttendanceDate:    d.AttendanceDate,
		Status:            string(d.Status),
		WorkDurationIndex: d.WorkDurationIndex,
		DailyRate:         d.DailyRate,
		TotalAmount:       d.TotalAmount,
		Notes:             d.Notes,
		CreatedByID:       d.CreatedByID,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}
}
