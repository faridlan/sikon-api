package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type customerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) domain.CustomerRepository {
	return &customerRepository{
		db: db,
	}
}

func (r *customerRepository) Create(ctx context.Context, customer *domain.Customer) error {
	model := FromCustomerDomain(customer)

	query := r.db.WithContext(ctx)

	if model.CreatedBy == "" {
		query = query.Omit("created_by")
	}

	err := query.Create(model).Error
	if err != nil {
		return TranslateError(err)
	}

	customer.ID = model.ID
	customer.CreatedAt = model.CreatedAt
	customer.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *customerRepository) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	var model CustomerModel

	err := r.db.WithContext(ctx).Preload("Creator").Where("id = ?", id).First(&model).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *customerRepository) Fetch(ctx context.Context, limit, offset int) ([]domain.Customer, int64, error) {
	var models []CustomerModel
	var total int64

	if err := r.db.WithContext(ctx).Model(&CustomerModel{}).Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	if err := r.db.WithContext(ctx).Preload("Creator").Limit(limit).Offset(offset).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	customers := make([]domain.Customer, len(models))
	for i, model := range models {
		customers[i] = *model.ToDomain()
	}

	return customers, total, nil
}

func (r *customerRepository) Update(ctx context.Context, customer *domain.Customer) error {
	model := FromCustomerDomain(customer)

	err := r.db.WithContext(ctx).Model(&CustomerModel{ID: model.ID}).Updates(model).Error
	if err != nil {
		return TranslateError(err)
	}

	return nil
}

func (r *customerRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&CustomerModel{}).Error
	if err != nil {
		return TranslateError(err)
	}

	return nil
}
