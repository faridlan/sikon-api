package postgres

import (
	"context"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type batchPoRepository struct {
	db *gorm.DB
}

func NewBatchPORepository(db *gorm.DB) domain.BatchPORepository {
	return &batchPoRepository{db: db}
}

func (r *batchPoRepository) Create(ctx context.Context, batchPO *domain.BatchPO) error {
	model := FromBatchPODomain(batchPO)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	batchPO.ID = model.ID
	batchPO.CreatedAt = model.CreatedAt
	batchPO.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *batchPoRepository) GetByID(ctx context.Context, id string) (*domain.BatchPO, error) {
	var model BatchPOModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *batchPoRepository) Fetch(ctx context.Context, limit, offset int) ([]domain.BatchPO, int64, error) {
	var models []BatchPOModel
	var total int64

	if err := r.db.WithContext(ctx).Model(&BatchPOModel{}).Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Order("start_date DESC").Find(&models).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	batchPOs := make([]domain.BatchPO, len(models))
	for i, model := range models {
		batchPOs[i] = *model.ToDomain()
	}
	return batchPOs, total, nil
}

func (r *batchPoRepository) FetchActive(ctx context.Context) ([]domain.BatchPO, error) {
	var models []BatchPOModel
	// Hanya ambil yang berstatus active untuk dropdown Frontend
	if err := r.db.WithContext(ctx).Where("status = ?", domain.BatchPOStatusActive).Order("start_date DESC").Find(&models).Error; err != nil {
		return nil, TranslateError(err)
	}

	batchPOs := make([]domain.BatchPO, len(models))
	for i, model := range models {
		batchPOs[i] = *model.ToDomain()
	}
	return batchPOs, nil
}

func (r *batchPoRepository) Update(ctx context.Context, batchPO *domain.BatchPO) error {
	model := FromBatchPODomain(batchPO)
	if err := r.db.WithContext(ctx).Model(&BatchPOModel{ID: model.ID}).Updates(model).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *batchPoRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&BatchPOModel{}).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *batchPoRepository) GetActivePOByDate(ctx context.Context, targetDate time.Time) (*domain.BatchPO, error) {
	var model BatchPOModel
	err := r.db.WithContext(ctx).
		Where("? BETWEEN start_date AND end_date", targetDate).
		Where("deleted_at IS NULL").
		Order("created_at DESC"). // Ambil yang paling baru dibuat jika ada overlap
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}
