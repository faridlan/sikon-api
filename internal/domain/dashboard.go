package domain

import "context"

// DashboardSummary merepresentasikan data agregasi untuk laporan halaman depan
type DashboardSummary struct {
	TotalRevenue         float64 // Total nilai pesanan (Grand Total dari order yang tidak batal)
	TotalPaymentReceived float64 // Total uang riil yang sudah masuk ke kas (Dari tabel payments)
	TotalReceivable      float64 // Total piutang (Sisa tagihan yang belum dibayar customer)

	TotalActiveOrders    int64 // Jumlah pesanan yang sedang berjalan (Status: pending & production)
	TotalCompletedOrders int64 // Jumlah pesanan yang sudah selesai dikerjakan
	TotalCanceledOrders  int64 // Jumlah pesanan yang dibatalkan
}

// DashboardFilter untuk menyaring laporan berdasarkan rentang waktu (harian, bulanan, tahunan)
type DashboardFilter struct {
	StartDate string // Format: YYYY-MM-DD
	EndDate   string // Format: YYYY-MM-DD
}

// DashboardRepository adalah kontrak untuk layer database (GORM)
type DashboardRepository interface {
	GetSummary(ctx context.Context, filter DashboardFilter) (*DashboardSummary, error)
}

// DashboardUsecase adalah kontrak untuk layer logika bisnis
type DashboardUsecase interface {
	GetSummary(ctx context.Context, filter DashboardFilter) (*DashboardSummary, error)
}
