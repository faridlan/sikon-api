package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

// --- IMPLEMENTASI REPOSITORY ---
type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) domain.OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, order *domain.Order) error {
	model := FromOrderDomain(order)

	// Memulai Database Transaction
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// tx.Create pada GORM secara otomatis akan menyimpan data induk (Order)
		// beserta relasinya (Items) jika struct-nya sudah terisi.
		if err := tx.Create(model).Error; err != nil {
			return err // Return error ini akan otomatis men-trigger ROLLBACK
		}

		// Jika ada logika lain (misal potong stok), bisa ditambahkan di dalam blok Transaction ini
		// ...

		return nil // Return nil berarti COMMIT (simpan permanen)
	})

	if err != nil {
		return TranslateError(err)
	}

	// Kembalikan ID yang di-generate ke layer domain
	order.ID = model.ID
	order.CreatedAt = model.CreatedAt
	order.UpdatedAt = model.UpdatedAt

	// Mengisi ID untuk setiap item yang baru dibuat
	for i := range model.Items {
		order.Items[i].ID = model.Items[i].ID
		order.Items[i].OrderID = model.ID
	}

	return nil
}

func (r *orderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	var model OrderModel

	// Kita Preload seluruh relasi penting agar struk invoice/nota nanti lengkap
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Customer").
		Preload("Sales").
		Preload("Items.Product"). // Preload produk di setiap item untuk detail yang lengkap
		Where("id = ?", id).
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *orderRepository) Fetch(ctx context.Context, filter domain.OrderFilter, limit, offset int) ([]domain.Order, int64, error) {
	var models []OrderModel
	var total int64

	// 1. Inisiasi instance GORM Model
	query := r.db.WithContext(ctx).Model(&OrderModel{})

	// 2. Terapkan Filter Dinamis (Jika nilainya ada)
	if filter.Search != "" {
		// Menggunakan ILIKE untuk pencarian case-insensitive di Postgres
		query = query.Where("order_number ILIKE ?", "%"+filter.Search+"%")
	}
	if filter.CustomerID != "" {
		query = query.Where("customer_id = ?", filter.CustomerID)
	}
	if filter.SalesID != "" {
		query = query.Where("sales_id = ?", filter.SalesID)
	}
	if filter.OrderStatus != "" {
		query = query.Where("order_status = ?", filter.OrderStatus)
	}
	if filter.PaymentStatus != "" {
		query = query.Where("payment_status = ?", filter.PaymentStatus)
	}
	if filter.StartDate != "" {
		// Mulai dari jam 00:00:00
		query = query.Where("created_at >= ?", filter.StartDate+" 00:00:00")
	}
	if filter.EndDate != "" {
		// Sampai jam 23:59:59
		query = query.Where("created_at <= ?", filter.EndDate+" 23:59:59")
	}

	// 3. Hitung Total Data (Penting! Harus dipanggil SETELAH filter diterapkan, tapi SEBELUM limit & offset)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	// 4. Ambil Data dengan Pagination dan Preload
	err := query.
		Preload("Customer"). // Hanya Preload relasi utama untuk list agar ringan
		Preload("Sales").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, 0, TranslateError(err)
	}

	orders := make([]domain.Order, len(models))
	for i, model := range models {
		orders[i] = *model.ToDomain()
	}

	return orders, total, nil
}

func (r *orderRepository) Update(ctx context.Context, order *domain.Order) error {
	model := FromOrderDomain(order)

	// Omit("Items") digunakan agar GORM tidak mencoba melakukan update/insert ulang pada order_items
	// Jika ingin mengupdate items, biasanya dibuatkan endpoint khusus atau logika tersendiri
	err := r.db.WithContext(ctx).Model(&OrderModel{ID: model.ID}).Omit("Items").Updates(model).Error
	if err != nil {
		return TranslateError(err)
	}

	return nil
}

func (r *orderRepository) Delete(ctx context.Context, id string) error {
	// Karena di file SQL Migration kita sudah menset "ON DELETE CASCADE" pada order_items,
	// menghapus Order di sini akan otomatis menghapus order_items yang terkait di database.
	err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&OrderModel{}).Error
	if err != nil {
		return TranslateError(err)
	}

	return nil
}

// Custom Query: Update Status secara spesifik
func (r *orderRepository) UpdateStatus(ctx context.Context, id string, orderStatus domain.OrderStatus, paymentStatus domain.PaymentStatus) error {
	updates := map[string]interface{}{}

	if orderStatus != "" {
		updates["order_status"] = string(orderStatus)
	}
	if paymentStatus != "" {
		updates["payment_status"] = string(paymentStatus)
	}

	if len(updates) == 0 {
		return nil // Tidak ada yang diupdate
	}

	err := r.db.WithContext(ctx).Model(&OrderModel{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		return TranslateError(err)
	}

	return nil
}
