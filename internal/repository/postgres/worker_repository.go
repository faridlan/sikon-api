package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type workerRepository struct {
	db *gorm.DB
}

func NewWorkerRepository(db *gorm.DB) domain.WorkerRepository {
	return &workerRepository{db: db}
}

func (r *workerRepository) Create(ctx context.Context, worker *domain.Worker) error {
	model := FromWorkerDomain(worker)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	worker.ID = model.ID
	worker.CreatedAt = model.CreatedAt
	worker.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *workerRepository) GetByID(ctx context.Context, id string) (*domain.Worker, error) {
	var model WorkerModel
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("id = ?", id).
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

// 🚨 TAMBAHAN METHOD: Ambil Worker berdasarkan UserID (untuk User yang login)
func (r *workerRepository) GetByUserID(ctx context.Context, userID string) (*domain.Worker, error) {
	var model WorkerModel
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("user_id = ?", userID).
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *workerRepository) Fetch(ctx context.Context, filter domain.WorkerFilter, limit, offset int) ([]domain.Worker, int64, error) {
	var models []WorkerModel
	var total int64

	query := r.db.WithContext(ctx).Model(&WorkerModel{})

	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR phone ILIKE ?", searchPattern, searchPattern)
	}

	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}

	if filter.SalaryType != "" {
		query = query.Where("salary_type = ?", filter.SalaryType)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.UserID != nil && *filter.UserID != "" {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	err := query.
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error

	if err != nil {
		return nil, 0, TranslateError(err)
	}

	var results []domain.Worker
	for _, m := range models {
		results = append(results, *m.ToDomain())
	}

	return results, total, nil
}

func (r *workerRepository) Update(ctx context.Context, worker *domain.Worker) error {
	model := FromWorkerDomain(worker)

	result := r.db.WithContext(ctx).
		Model(&WorkerModel{}).
		Where("id = ?", worker.ID).
		Updates(map[string]interface{}{
			"user_id":     model.UserID,
			"name":        model.Name,
			"phone":       model.Phone,
			"role":        model.Role,
			"salary_type": model.SalaryType,
			"daily_rate":  model.DailyRate,
			"status":      model.Status,
			"updated_at":  model.UpdatedAt,
		})

	if result.Error != nil {
		return TranslateError(result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *workerRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&WorkerModel{})
	if result.Error != nil {
		return TranslateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
