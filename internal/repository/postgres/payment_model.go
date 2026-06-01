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

	Order       *OrderModel       `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	BankAccount *BankAccountModel `gorm:"foreignKey:BankAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
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
		PaymentType:     domain.PaymentType(m.PaymentType), // Konversi kembali ke Enum (Custom Type)
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	// Map relasi Order jika tersedia
	if m.Order != nil {
		payment.Order = m.Order.ToDomain()
	}
	// Map relasi BankAccount jika tersedia
	if m.BankAccount != nil {
		payment.BankAccount = m.BankAccount.ToDomain()
	}

	return payment
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
