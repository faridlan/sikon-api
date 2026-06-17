package domain

import "context"

// TransactionManager adalah kontrak untuk memulai Kertas Buram (Transaksi).
// Siapapun yang mengimplementasikan ini (nantinya GORM),
// harus bisa menjalankan kumpulan perintah di dalam satu payung transaksi.
type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
