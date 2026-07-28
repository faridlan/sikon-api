package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

// ==========================================
// MODEL: EXPENSE CATEGORY
// ==========================================
type ExpenseCategoryModel struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name        string         `gorm:"type:varchar(255);not null"`
	Type        string         `gorm:"type:varchar(50);not null"`
	Description string         `gorm:"type:text"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (ExpenseCategoryModel) TableName() string {
	return "expense_categories"
}

func (m *ExpenseCategoryModel) ToDomain() *domain.ExpenseCategory {
	return &domain.ExpenseCategory{
		ID:          m.ID,
		Name:        m.Name,
		Type:        domain.ExpenseCategoryType(m.Type),
		Description: m.Description,
	}
}

func FromExpenseCategoryDomain(d *domain.ExpenseCategory) *ExpenseCategoryModel {
	return &ExpenseCategoryModel{
		ID:          d.ID,
		Name:        d.Name,
		Type:        string(d.Type),
		Description: d.Description,
	}
}

// ==========================================
// MODEL: EXPENSE TRANSACTION
// ==========================================
type ExpenseModel struct {
	ID                string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ExpenseCategoryID string         `gorm:"type:uuid;not null;index"`
	BatchPoID         *string        `gorm:"type:uuid;index"`
	Title             string         `gorm:"type:varchar(255);not null"`
	Amount            float64        `gorm:"type:numeric(15,2);not null"`
	ExpenseDate       time.Time      `gorm:"type:date;not null;index"`
	Notes             string         `gorm:"type:text"`
	CreatedByID       string         `gorm:"type:uuid;not null"`
	CreatedAt         time.Time      `gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`

	// Relasi
	Category ExpenseCategoryModel `gorm:"foreignKey:ExpenseCategoryID"`
	BatchPO  BatchPOModel         `gorm:"foreignKey:BatchPoID"`
	Creator  UserModel            `gorm:"foreignKey:CreatedByID"`
}

func (ExpenseModel) TableName() string {
	return "expenses"
}

func (m *ExpenseModel) ToDomain() *domain.Expense {
	exp := &domain.Expense{
		ID:                m.ID,
		ExpenseCategoryID: m.ExpenseCategoryID,
		CategoryName:      m.Category.Name,
		CategoryType:      m.Category.Type,
		BatchPoID:         m.BatchPoID,
		Title:             m.Title,
		Amount:            m.Amount,
		ExpenseDate:       m.ExpenseDate,
		Notes:             m.Notes,
		CreatedByID:       m.CreatedByID,
		CreatorName:       m.Creator.Name,
		CreatedAt:         m.CreatedAt,
	}

	// Jika BatchPO di-preload dan datanya ada
	if m.BatchPO.ID != "" {
		exp.BatchPoName = &m.BatchPO.Name
	}

	return exp
}

func FromExpenseDomain(d *domain.Expense) *ExpenseModel {
	return &ExpenseModel{
		ID:                d.ID,
		ExpenseCategoryID: d.ExpenseCategoryID,
		BatchPoID:         d.BatchPoID,
		Title:             d.Title,
		Amount:            d.Amount,
		ExpenseDate:       d.ExpenseDate,
		Notes:             d.Notes,
		CreatedByID:       d.CreatedByID,
	}
}
