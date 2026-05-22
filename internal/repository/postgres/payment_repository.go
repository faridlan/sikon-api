package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) domain.PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	model := FromPaymentDomain(payment)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	payment.ID = model.ID
	payment.CreatedAt = model.CreatedAt
	payment.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *paymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	var model PaymentModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *paymentRepository) Fetch(ctx context.Context, limit, offset int) ([]domain.Payment, int64, error) {
	var models []PaymentModel
	var total int64

	if err := r.db.WithContext(ctx).Model(&PaymentModel{}).Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Order("payment_date DESC").Find(&models).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	payments := make([]domain.Payment, len(models))
	for i, model := range models {
		payments[i] = *model.ToDomain()
	}
	return payments, total, nil
}

func (r *paymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	model := FromPaymentDomain(payment)
	if err := r.db.WithContext(ctx).Model(&PaymentModel{ID: model.ID}).Updates(model).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *paymentRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&PaymentModel{}).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID string) ([]domain.Payment, error) {
	var models []PaymentModel

	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("payment_date ASC").Find(&models).Error; err != nil {
		return nil, TranslateError(err)
	}

	payments := make([]domain.Payment, len(models))
	for i, model := range models {
		payments[i] = *model.ToDomain()
	}
	return payments, nil
}
