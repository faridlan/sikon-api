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

	// TAMBAHAN: Tambahkan .Preload("Sales") agar data user(sales) ikut terbawa
	err := r.db.WithContext(ctx).Preload("Creator").Preload("Sales").Where("id = ?", id).First(&model).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *customerRepository) Fetch(ctx context.Context, filter domain.CustomerFilter, limit, offset int) ([]domain.Customer, int64, error) {
	var models []CustomerModel
	var total int64

	// 1. Mulai query builder
	query := r.db.WithContext(ctx).Model(&CustomerModel{})

	// 2. Filter Pencarian Teks (Nama atau Phone)
	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		// Menggunakan kurung () di dalam SQL agar OR tidak bocor ke logika AND lainnya
		query = query.Where("(name ILIKE ? OR phone ILIKE ?)", searchTerm, searchTerm)
	}

	// 3. Filter Sales ID (Sudah aman karena disaring di Usecase)
	if filter.SalesID != "" {
		query = query.Where("sales_id = ?", filter.SalesID)
	}

	// 4. Hitung total data sesuai filter
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	// 5. Eksekusi Pagination dan Preload
	err := query.
		Preload("Creator").
		Preload("Sales").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&models).Error

	if err != nil {
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
