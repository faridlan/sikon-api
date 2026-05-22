package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type PaymentModel struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderID         string    `gorm:"type:uuid;not null"`
	BankAccountID   string    `gorm:"type:uuid;not null"`
	Amount          float64   `gorm:"type:decimal(15,2);not null;default:0"`
	PaymentDate     time.Time `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP"`
	ReferenceNumber string    `gorm:"type:varchar(100)"`
	PaymentType     string    `gorm:"type:varchar(50);not null"` // "dp", "settlement", "installment"
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (PaymentModel) TableName() string {
	return "payments"
}

func (m *PaymentModel) ToDomain() *domain.Payment {
	return &domain.Payment{
		ID:              m.ID,
		OrderID:         m.OrderID,
		BankAccountID:   m.BankAccountID,
		Amount:          m.Amount,
		PaymentDate:     m.PaymentDate,
		ReferenceNumber: m.ReferenceNumber,
		PaymentType:     domain.PaymentType(m.PaymentType), // Konversi kembali ke Enum (Custom Type)
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func FromPaymentDomain(d *domain.Payment) *PaymentModel {
	return &PaymentModel{
		ID:              d.ID,
		OrderID:         d.OrderID,
		BankAccountID:   d.BankAccountID,
		Amount:          d.Amount,
		PaymentDate:     d.PaymentDate,
		ReferenceNumber: d.ReferenceNumber,
		PaymentType:     string(d.PaymentType),
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}
