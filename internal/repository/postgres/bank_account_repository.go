package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type bankAccountRepository struct {
	db *gorm.DB
}

func NewBankAccountRepository(db *gorm.DB) domain.BankAccountRepository {
	return &bankAccountRepository{db: db}
}

func (r *bankAccountRepository) Create(ctx context.Context, account *domain.BankAccount) error {
	model := FromBankAccountDomain(account)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	account.ID = model.ID
	account.CreatedAt = model.CreatedAt
	account.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *bankAccountRepository) GetByID(ctx context.Context, id string) (*domain.BankAccount, error) {
	var model BankAccountModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *bankAccountRepository) Fetch(ctx context.Context, limit, offset int) ([]domain.BankAccount, int64, error) {
	var models []BankAccountModel
	var total int64

	if err := r.db.WithContext(ctx).Model(&BankAccountModel{}).Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	accounts := make([]domain.BankAccount, len(models))
	for i, model := range models {
		accounts[i] = *model.ToDomain()
	}
	return accounts, total, nil
}

func (r *bankAccountRepository) Update(ctx context.Context, account *domain.BankAccount) error {
	model := FromBankAccountDomain(account)
	if err := r.db.WithContext(ctx).Model(&BankAccountModel{ID: model.ID}).Updates(model).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *bankAccountRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&BankAccountModel{}).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

// Custom Query: Mengambil rekening berdasarkan UserID (Sales) atau Rekening Global (UserID is null)
func (r *bankAccountRepository) GetByUserID(ctx context.Context, userID string) ([]domain.BankAccount, error) {
	var models []BankAccountModel
	// Jika userID kosong, ambil rekening global (perusahaan)
	query := r.db.WithContext(ctx)
	if userID == "" {
		query = query.Where("user_id IS NULL")
	} else {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Find(&models).Error; err != nil {
		return nil, TranslateError(err)
	}

	accounts := make([]domain.BankAccount, len(models))
	for i, model := range models {
		accounts[i] = *model.ToDomain()
	}
	return accounts, nil
}

// Tambahan implementasi GetGlobalAccounts (karena ada di interface domain)
func (r *bankAccountRepository) GetGlobalAccounts(ctx context.Context) ([]domain.BankAccount, error) {
	return r.GetByUserID(ctx, "")
}
