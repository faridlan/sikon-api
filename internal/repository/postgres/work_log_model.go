package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type WorkLogModel struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkerID    string         `gorm:"type:uuid;not null"`
	BatchPoID   *string        `gorm:"type:uuid"`
	OrderID     *string        `gorm:"type:uuid"` // 👈 TAMBAHKAN KOLOML INI
	PayrollID   *string        `gorm:"type:uuid"`
	JobType     string         `gorm:"type:varchar(50);not null"`
	Qty         int            `gorm:"not null;default:1"`
	RatePerQty  float64        `gorm:"type:numeric(15,2);not null;default:0"`
	TotalAmount float64        `gorm:"type:numeric(15,2);not null;default:0"`
	WorkDate    time.Time      `gorm:"type:date;not null"`
	Notes       string         `gorm:"type:text"`
	CreatedByID string         `gorm:"column:created_by;type:uuid;not null"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Relasi (Preload)
	Worker  *WorkerModel  `gorm:"foreignKey:WorkerID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	BatchPO *BatchPOModel `gorm:"foreignKey:BatchPoID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Order   *OrderModel   `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"` // 👈 RELASI ORDER
	Payroll *PayrollModel `gorm:"foreignKey:PayrollID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Creator *UserModel    `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

func (WorkLogModel) TableName() string {
	return "work_logs"
}

func (m *WorkLogModel) ToDomain() *domain.WorkLog {
	log := &domain.WorkLog{
		ID:          m.ID,
		WorkerID:    m.WorkerID,
		BatchPoID:   m.BatchPoID,
		OrderID:     m.OrderID, // 👈 MAPPER TO DOMAIN
		PayrollID:   m.PayrollID,
		JobType:     domain.JobType(m.JobType),
		Qty:         m.Qty,
		RatePerQty:  m.RatePerQty,
		TotalAmount: m.TotalAmount,
		WorkDate:    m.WorkDate,
		Notes:       m.Notes,
		CreatedByID: m.CreatedByID,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}

	if m.Worker != nil {
		log.Worker = m.Worker.ToDomain()
	}

	if m.BatchPO != nil {
		log.BatchPO = m.BatchPO.ToDomain()
	}

	if m.Order != nil {
		log.Order = m.Order.ToDomain() // 👈 PRELOAD ORDER
	}

	if m.Creator != nil {
		log.Creator = m.Creator.ToDomain()
	}

	return log
}

func FromWorkLogDomain(d *domain.WorkLog) *WorkLogModel {
	return &WorkLogModel{
		ID:          d.ID,
		WorkerID:    d.WorkerID,
		BatchPoID:   d.BatchPoID,
		OrderID:     d.OrderID, // 👈 MAPPER FROM DOMAIN
		PayrollID:   d.PayrollID,
		JobType:     string(d.JobType),
		Qty:         d.Qty,
		RatePerQty:  d.RatePerQty,
		TotalAmount: d.TotalAmount,
		WorkDate:    d.WorkDate,
		Notes:       d.Notes,
		CreatedByID: d.CreatedByID,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
