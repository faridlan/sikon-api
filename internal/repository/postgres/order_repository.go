package postgres

import (
	"context"
	"encoding/json"

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

	db := GetTx(ctx, r.db) // Ambil DB dari Context (bisa jadi transaction)

	if err := db.WithContext(ctx).Create(&model).Error; err != nil {
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
	err := GetTx(ctx, r.db).
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

	// 🚨 PERUBAHAN DI SINI 🚨
	db := GetTx(ctx, r.db)

	err := db.WithContext(ctx).Model(&OrderModel{ID: model.ID}).Omit("Items").Updates(model).Error
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

func (r *orderRepository) UpdateStatus(ctx context.Context, id string, orderStatus domain.OrderStatus, paymentStatus domain.PaymentStatus) error {
	updates := map[string]any{}

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

func (r *orderRepository) GetItemByID(ctx context.Context, orderID, itemID string) (*domain.OrderItem, error) {
	var item domain.OrderItem
	// Kita langsung query ke tabel order_items
	err := r.db.WithContext(ctx).Table("order_items").
		Where("order_id = ? AND id = ?", orderID, itemID).
		First(&item).Error

	if err != nil {
		return nil, TranslateError(err)
	}
	return &item, nil
}

func (r *orderRepository) CreateItem(ctx context.Context, item *domain.OrderItem) error {
	var detailsJSON []byte
	if item.Details != nil {
		detailsJSON, _ = json.Marshal(item.Details)
	}

	// 🚨 PERUBAHAN DI SINI 🚨
	// Staf mengecek: Ada Kertas Buram nggak di Context?
	// Kalau ada, pakai itu. Kalau tidak ada, pakai r.db biasa.
	db := GetTx(ctx, r.db)

	err := db.WithContext(ctx).Table("order_items").Create(map[string]any{
		"id":         item.ID,
		"order_id":   item.OrderID,
		"product_id": item.ProductID,
		"qty":        item.Qty,
		"price":      item.Price,
		"details":    detailsJSON,
	}).Error

	if err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *orderRepository) UpdateItem(ctx context.Context, item *domain.OrderItem) error {

	var detailsJSON []byte
	if item.Details != nil {
		detailsJSON, _ = json.Marshal(item.Details)
	}

	// Note: Menggunakan map alih-alih OrderItemModel struct untuk menghindari
	// GORM secara tidak sengaja meng-update tabel relasi (Product/Order)
	// dan memastikan zero-value (seperti qty/harga 0) tetap ter-update.
	err := r.db.WithContext(ctx).Table("order_items").
		Where("id = ? AND order_id = ?", item.ID, item.OrderID).
		Updates(map[string]any{
			"product_id": item.ProductID,
			"qty":        item.Qty,
			"price":      item.Price,
			"details":    detailsJSON,
		}).Error

	if err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *orderRepository) DeleteItem(ctx context.Context, orderID, itemID string) error {
	err := r.db.WithContext(ctx).Table("order_items").
		Where("order_id = ? AND id = ?", orderID, itemID).
		Delete(nil).Error // Delete record

	if err != nil {
		return TranslateError(err)
	}
	return nil
}
