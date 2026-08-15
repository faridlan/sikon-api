package postgres

import (
	"context"
	"time"

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
	if payment.Status == "" {
		payment.Status = domain.PaymentVerificationPending
	}

	model := FromPaymentDomain(payment)
	db := GetTx(ctx, r.db)

	if err := db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	payment.ID = model.ID
	payment.CreatedAt = model.CreatedAt
	payment.UpdatedAt = model.UpdatedAt
	payment.Status = domain.PaymentVerificationStatus(model.Status)

	return nil
}

func (r *paymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	var model PaymentModel
	db := GetTx(ctx, r.db)

	if err := db.WithContext(ctx).Preload("Order").Preload("BankAccount").Where("id = ?", id).First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *paymentRepository) Fetch(ctx context.Context, limit, offset int, filter domain.PaymentFilter) ([]domain.Payment, int64, error) {
	var models []PaymentModel
	var total int64

	db := GetTx(ctx, r.db)
	query := db.WithContext(ctx).Model(&PaymentModel{}).
		Joins("LEFT JOIN orders ON orders.id = payments.order_id")

	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("payments.reference_number ILIKE ? OR orders.order_number ILIKE ?", searchTerm, searchTerm)
	}
	if filter.PaymentType != "" {
		query = query.Where("payments.payment_type = ?", filter.PaymentType)
	}
	if filter.Status != "" {
		query = query.Where("payments.status = ?", filter.Status)
	}
	if filter.SalesID != "" {
		query = query.Where("orders.sales_id = ?", filter.SalesID)
	}
	if filter.StartDate != "" {
		query = query.Where("payments.payment_date >= ?", filter.StartDate+" 00:00:00")
	}
	if filter.EndDate != "" {
		query = query.Where("payments.payment_date <= ?", filter.EndDate+" 23:59:59")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	err := query.
		Preload("Order").
		Preload("BankAccount").
		Preload("VerifiedBy").
		Limit(limit).
		Offset(offset).
		Order("payments.created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, 0, TranslateError(err)
	}

	payments := make([]domain.Payment, len(models))
	for i, m := range models {
		payments[i] = *m.ToDomain()
	}

	return payments, total, nil
}

func (r *paymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	db := GetTx(ctx, r.db)

	// Menggunakan Map untuk menghindari jebakan Zero Value GORM Struct
	updates := map[string]any{
		"reference_number": payment.ReferenceNumber,
		"payment_type":     string(payment.PaymentType),
		"amount":           payment.Amount,
		"bank_account_id":  payment.BankAccountID,
		"payment_date":     payment.PaymentDate,
		"updated_at":       time.Now(),
	}

	if err := db.WithContext(ctx).Table("payments").Where("id = ? AND deleted_at IS NULL", payment.ID).Updates(updates).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *paymentRepository) Delete(ctx context.Context, id string) error {
	db := GetTx(ctx, r.db)

	if err := db.WithContext(ctx).Where("id = ?", id).Delete(&PaymentModel{}).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID string) ([]domain.Payment, error) {
	var models []PaymentModel
	db := GetTx(ctx, r.db)

	err := db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at ASC").
		Find(&models).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	payments := make([]domain.Payment, len(models))
	for i, m := range models {
		payments[i] = *m.ToDomain()
	}

	return payments, nil
}

// 🚨 PERBAIKAN RACE CONDITION & DOUBLE VERIFICATION
func (r *paymentRepository) UpdateVerificationStatus(ctx context.Context, paymentID string, status domain.PaymentVerificationStatus, verifiedByID string, verifiedAt time.Time) error {
	var verifiedByVal any = verifiedByID
	if verifiedByID == "" {
		verifiedByVal = nil
	}

	updates := map[string]any{
		"status":         string(status),
		"verified_by_id": verifiedByVal,
		"verified_at":    verifiedAt,
		"updated_at":     time.Now(),
	}

	db := GetTx(ctx, r.db)

	// Guarding: Kunci hanya pembayaran berstatus 'pending' yang boleh di-update
	res := db.WithContext(ctx).Table("payments").
		Where("id = ? AND status = ? AND deleted_at IS NULL", paymentID, domain.PaymentVerificationPending).
		Updates(updates)

	if res.Error != nil {
		return TranslateError(res.Error)
	}

	// Jika RowsAffected == 0, artinya status sudah di-verify/reject sebelumnya (mencegah double execution)
	if res.RowsAffected == 0 {
		return domain.NewError(domain.ErrConflict, "Pembayaran ini sudah pernah diverifikasi atau diproses sebelumnya")
	}

	return nil
}
