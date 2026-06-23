package postgres

import (
	"context"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type dashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) domain.DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) GetSummary(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error) {
	summary := &domain.DashboardSummary{}

	// 1. Setup Query Base (Menambahkan Filter Tanggal Jika Ada)
	orderQuery := r.db.WithContext(ctx).Model(&OrderModel{})
	paymentQuery := r.db.WithContext(ctx).Model(&PaymentModel{})

	if filter.StartDate != "" && filter.EndDate != "" {
		// Konversi string YYYY-MM-DD ke tipe data time.Time
		start, errStart := time.Parse("2006-01-02", filter.StartDate)
		end, errEnd := time.Parse("2006-01-02", filter.EndDate)

		if errStart == nil && errEnd == nil {
			// Set waktu akhir ke pukul 23:59:59 agar meng-cover transaksi di hari tersebut sepenuhnya
			end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

			orderQuery = orderQuery.Where("created_at BETWEEN ? AND ?", start, end)
			// Untuk payment, kita asumsikan difilter berdasarkan PaymentDate
			paymentQuery = paymentQuery.Where("payment_date BETWEEN ? AND ?", start, end)
		}
	}

	// 2. Eksekusi Kalkulasi Order (Hanya butuh 1x Hit ke Database untuk 4 Metrik!)
	// Menggunakan fungsi agregasi CASE WHEN di PostgreSQL untuk memilah data secara instan
	type OrderStats struct {
		TotalRevenue    float64
		ActiveOrders    int64
		CompletedOrders int64
		CanceledOrders  int64
	}
	var stats OrderStats

	err := orderQuery.Select(`
		COALESCE(SUM(CASE WHEN order_status != 'canceled' THEN total_amount ELSE 0 END), 0) as total_revenue,
		COALESCE(SUM(CASE WHEN order_status IN ('pending', 'production') THEN 1 ELSE 0 END), 0) as active_orders,
		COALESCE(SUM(CASE WHEN order_status = 'completed' THEN 1 ELSE 0 END), 0) as completed_orders,
		COALESCE(SUM(CASE WHEN order_status = 'canceled' THEN 1 ELSE 0 END), 0) as canceled_orders
	`).Scan(&stats).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	summary.TotalRevenue = stats.TotalRevenue
	summary.TotalActiveOrders = stats.ActiveOrders
	summary.TotalCompletedOrders = stats.CompletedOrders
	summary.TotalCanceledOrders = stats.CanceledOrders

	// 3. Eksekusi Kalkulasi Total Uang Masuk (Berdasarkan filter tanggal)
	err = paymentQuery.Select("COALESCE(SUM(amount), 0)").Scan(&summary.TotalPaymentReceived).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	// 4. Eksekusi Kalkulasi Piutang / Uang Menggantung (Total Receivable)
	// CATATAN BISNIS: Piutang adalah "Saldo" (Snapshot), jadi TIDAK BOLEH terkena filter tanggal.
	// Piutang = Total Omzet Sepanjang Masa - Total Pembayaran Sepanjang Masa
	var allTimeRevenue float64
	var allTimePayment float64

	r.db.WithContext(ctx).Model(&OrderModel{}).
		Where("order_status != 'canceled'").
		Select("COALESCE(SUM(total_amount), 0)").Scan(&allTimeRevenue)

	r.db.WithContext(ctx).Model(&PaymentModel{}).
		Select("COALESCE(SUM(amount), 0)").Scan(&allTimePayment)

	summary.TotalReceivable = allTimeRevenue - allTimePayment
	if summary.TotalReceivable < 0 {
		summary.TotalReceivable = 0 // Mencegah angka minus jika ada overpayment atau diskon
	}

	return summary, nil
}
