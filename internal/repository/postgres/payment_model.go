package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type PaymentModel struct {
	ID              string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderID         string         `gorm:"type:uuid;not null"`
	BankAccountID   string         `gorm:"type:uuid;not null"`
	Amount          float64        `gorm:"type:decimal(15,2);not null;default:0"`
	PaymentDate     time.Time      `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP"`
	ReferenceNumber string         `gorm:"type:varchar(100)"`
	PaymentType     string         `gorm:"type:varchar(50);not null"`
	Status          string         `gorm:"type:varchar(50);not null;default:'pending'"` // BARU
	VerifiedByID    *string        `gorm:"type:uuid"`                                   // BARU
	VerifiedAt      *time.Time     `gorm:"type:timestamp with time zone"`               // BARU
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	Order       *OrderModel       `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	BankAccount *BankAccountModel `gorm:"foreignKey:BankAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	VerifiedBy  *UserModel        `gorm:"foreignKey:VerifiedByID"`
}

func (PaymentModel) TableName() string {
	return "payments"
}

func (m *PaymentModel) ToDomain() *domain.Payment {
	payment := &domain.Payment{
		ID:              m.ID,
		OrderID:         m.OrderID,
		BankAccountID:   m.BankAccountID,
		Amount:          m.Amount,
		PaymentDate:     m.PaymentDate,
		ReferenceNumber: m.ReferenceNumber,
		PaymentType:     domain.PaymentType(m.PaymentType),
		Status:          domain.PaymentVerificationStatus(m.Status),
		VerifiedByID:    m.VerifiedByID,
		VerifiedAt:      m.VerifiedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	if m.Order != nil {
		payment.Order = m.Order.ToDomain()
	}
	if m.BankAccount != nil {
		payment.BankAccount = m.BankAccount.ToDomain()
	}
	if m.VerifiedBy != nil {
		payment.VerifiedBy = m.VerifiedBy.ToDomain()
	}

	return payment
}

func FromPaymentDomain(d *domain.Payment) *PaymentModel {
	status := string(d.Status)
	if status == "" {
		status = string(domain.PaymentVerificationPending)
	}

	return &PaymentModel{
		ID:              d.ID,
		OrderID:         d.OrderID,
		BankAccountID:   d.BankAccountID,
		Amount:          d.Amount,
		PaymentDate:     d.PaymentDate,
		ReferenceNumber: d.ReferenceNumber,
		PaymentType:     string(d.PaymentType),
		Status:          status,
		VerifiedByID:    d.VerifiedByID,
		VerifiedAt:      d.VerifiedAt,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}
