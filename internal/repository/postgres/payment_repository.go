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

	db := GetTx(ctx, r.db)

	if err := db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	payment.ID = model.ID
	payment.CreatedAt = model.CreatedAt
	payment.UpdatedAt = model.UpdatedAt

	return nil
}

func (r *paymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	var model PaymentModel
	if err := r.db.WithContext(ctx).Preload("Order").Preload("BankAccount").Where("id = ?", id).First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *paymentRepository) Fetch(ctx context.Context, limit, offset int, filter domain.PaymentFilter) ([]domain.Payment, int64, error) {
	var models []PaymentModel
	var total int64

	// Mulai query base
	query := r.db.WithContext(ctx).Model(&PaymentModel{})

	// 1. Filter Search (mencari substring di No Ref atau exact match di OrderID)
	if filter.Search != "" {
		searchParam := "%" + filter.Search + "%"
		// ::text digunakan di PostgreSQL untuk memungkinkan ILIKE pada tipe data UUID (OrderID)
		query = query.Where("reference_number ILIKE ? OR order_id::text ILIKE ?", searchParam, searchParam)
	}

	// 2. Filter Payment Type
	if filter.PaymentType != "" {
		query = query.Where("payment_type = ?", filter.PaymentType)
	}

	// 3. Filter Start Date (dari awal hari)
	if filter.StartDate != "" {
		query = query.Where("payment_date >= ?", filter.StartDate+" 00:00:00")
	}

	// 4. Filter End Date (hingga akhir hari)
	if filter.EndDate != "" {
		query = query.Where("payment_date <= ?", filter.EndDate+" 23:59:59")
	}

	// Hitung total data yang cocok dengan query
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	// Eksekusi pengambilan datanya
	if err := query.Preload("Order").Preload("BankAccount").
		Limit(limit).Offset(offset).Order("payment_date DESC").
		Find(&models).Error; err != nil {
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

	if err := r.db.WithContext(ctx).Preload("Order").Preload("BankAccount").Where("order_id = ?", orderID).Order("payment_date ASC").Find(&models).Error; err != nil {
		return nil, TranslateError(err)
	}

	payments := make([]domain.Payment, len(models))
	for i, model := range models {
		payments[i] = *model.ToDomain()
	}
	return payments, nil
}
