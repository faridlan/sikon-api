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

// FetchActive mengembalikan PO yang BENAR-BENAR sedang dalam jendela penerimaan order
// (open_date <= sekarang <= close_date) DAN belum di-nonaktifkan manual (status = active).
//
// 🚨 PERBAIKAN PENTING: sebelumnya query ini HANYA mengecek `status = active`, tanpa validasi
// tanggal sama sekali. Akibatnya kalau Admin lupa menutup PO lama (status-nya nyangkut 'active'),
// PO yang sudah lewat jendelanya tetap muncul sebagai pilihan default di FE — inilah penyebab
// bug "PO September 2 yang muncul, padahal harusnya PO September 3".
//
// Dengan filter tanggal ditambahkan, PO yang jendelanya sudah lewat otomatis tersingkir dari
// hasil query ini walау status-nya lupa di-update, sehingga kelas bug ini tidak bisa terulang.
func (r *batchPoRepository) FetchActive(ctx context.Context) ([]domain.BatchPO, error) {
	var models []BatchPOModel
	now := time.Now()

	err := r.db.WithContext(ctx).
		Where("status = ?", domain.BatchPOStatusActive).
		Where("open_date <= ? AND close_date >= ?", now, now).
		Order("start_date DESC").
		Find(&models).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	batchPOs := make([]domain.BatchPO, len(models))
	for i, model := range models {
		batchPOs[i] = *model.ToDomain()
	}
	return batchPOs, nil
}

// FetchLatest mengambil PO dengan close_date paling besar (paling baru jendelanya),
// dipakai untuk auto-prefill open_date saat Admin membuat PO baru di FE.
func (r *batchPoRepository) FetchLatest(ctx context.Context) (*domain.BatchPO, error) {
	var model BatchPOModel
	err := r.db.WithContext(ctx).
		Order("close_date DESC").
		First(&model).Error
	if err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
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

// GetActivePOByDate dipakai order_usecase.go saat Order di-approve, untuk menentukan
// PO mana yang harus jadi tujuan alokasi. Sama seperti FetchActive, sekarang berbasis
// open_date/close_date (jendela penerimaan order), BUKAN start_date/end_date (periode kerja).
func (r *batchPoRepository) GetActivePOByDate(ctx context.Context, targetDate time.Time) (*domain.BatchPO, error) {
	var model BatchPOModel
	err := r.db.WithContext(ctx).
		Where("? BETWEEN open_date AND close_date", targetDate).
		Where("status = ?", domain.BatchPOStatusActive).
		Where("deleted_at IS NULL").
		Order("created_at DESC"). // Ambil yang paling baru dibuat jika ada overlap
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}
