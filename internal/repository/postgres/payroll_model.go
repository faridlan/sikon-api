package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type PayrollModel struct {
	ID            string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PayrollNumber string         `gorm:"type:varchar(100);not null;unique"`
	StartDate     time.Time      `gorm:"type:date;not null"`
	EndDate       time.Time      `gorm:"type:date;not null"`
	TotalAmount   float64        `gorm:"type:numeric(15,2);not null;default:0"`
	Status        string         `gorm:"type:varchar(50);not null;default:'draft'"`
	ExpenseID     *string        `gorm:"type:uuid"`
	CreatedByID   string         `gorm:"column:created_by;type:uuid;not null"`
	PaidAt        *time.Time     `gorm:"type:timestamp with time zone"`
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relasi (Preload)
	Creator     *UserModel        `gorm:"foreignKey:CreatedByID"`
	Expense     *ExpenseModel     `gorm:"foreignKey:ExpenseID"`
	WorkLogs    []WorkLogModel    `gorm:"foreignKey:PayrollID"`
	Attendances []AttendanceModel `gorm:"foreignKey:PayrollID"` // 👈 Relasi ke Attendances
}

func (PayrollModel) TableName() string {
	return "payrolls"
}

func (m *PayrollModel) ToDomain() *domain.Payroll {
	payroll := &domain.Payroll{
		ID:            m.ID,
		PayrollNumber: m.PayrollNumber,
		StartDate:     m.StartDate,
		EndDate:       m.EndDate,
		TotalAmount:   m.TotalAmount,
		Status:        domain.PayrollStatus(m.Status),
		ExpenseID:     m.ExpenseID,
		CreatedByID:   m.CreatedByID,
		PaidAt:        m.PaidAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}

	if m.Creator != nil {
		payroll.Creator = m.Creator.ToDomain()
	}

	if m.Expense != nil {
		payroll.Expense = m.Expense.ToDomain()
	}

	if len(m.WorkLogs) > 0 {
		var logs []domain.WorkLog
		for _, wl := range m.WorkLogs {
			logs = append(logs, *wl.ToDomain())
		}
		payroll.WorkLogs = logs
	}

	if len(m.Attendances) > 0 {
		var atts []domain.Attendance
		for _, att := range m.Attendances {
			atts = append(atts, *att.ToDomain())
		}
		payroll.Attendances = atts
	}

	return payroll
}

func FromPayrollDomain(d *domain.Payroll) *PayrollModel {
	return &PayrollModel{
		ID:            d.ID,
		PayrollNumber: d.PayrollNumber,
		StartDate:     d.StartDate,
		EndDate:       d.EndDate,
		TotalAmount:   d.TotalAmount,
		Status:        string(d.Status),
		ExpenseID:     d.ExpenseID,
		CreatedByID:   d.CreatedByID,
		PaidAt:        d.PaidAt,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
