package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type BankAccountModel struct {
	ID            string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID        *string        `gorm:"type:uuid"` // Pointer karena bisa NULL (rekening global)
	BankName      string         `gorm:"type:varchar(100);not null"`
	AccountNumber string         `gorm:"type:varchar(100);not null"`
	AccountName   string         `gorm:"type:varchar(255);not null"`
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relasi
	User *UserModel `gorm:"foreignKey:UserID"`
}

func (BankAccountModel) TableName() string {
	return "bank_accounts"
}

func (m *BankAccountModel) ToDomain() *domain.BankAccount {

	bankAccount := &domain.BankAccount{
		ID:            m.ID,
		UserID:        m.UserID,
		BankName:      m.BankName,
		AccountNumber: m.AccountNumber,
		AccountName:   m.AccountName,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}

	if m.User != nil {
		bankAccount.User = m.User.ToDomain()
	}

	return bankAccount
}

func FromBankAccountDomain(d *domain.BankAccount) *BankAccountModel {
	return &BankAccountModel{
		ID:            d.ID,
		UserID:        d.UserID,
		BankName:      d.BankName,
		AccountNumber: d.AccountNumber,
		AccountName:   d.AccountName,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
