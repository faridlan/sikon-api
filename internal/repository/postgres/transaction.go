package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

// txKey adalah kunci rahasia untuk menyelipkan Kertas Buram ke dalam Map Dokumen (Context)
type txKey struct{}

type transactionManager struct {
	db *gorm.DB
}

// NewTransactionManager mencetak mesin pengelola transaksi
func NewTransactionManager(db *gorm.DB) domain.TransactionManager {
	return &transactionManager{db: db}
}

// RunInTransaction adalah proses dari awal ngambil Kertas Buram sampai mencatat ke Buku Besar
func (t *transactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// 1. Ambil Kertas Buram (Begin Transaction)
	tx := t.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Jaring pengaman: Jika tiba-tiba sistem crash (panic), otomatis buang Kertas Buram
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 2. Selipkan Kertas Buram (tx) ke dalam Map Dokumen (Context)
	txCtx := context.WithValue(ctx, txKey{}, tx)

	// 3. Suruh Mandor (fn / Usecase) bekerja membawa Map Dokumen yang baru
	err := fn(txCtx)
	if err != nil {
		// Jika Mandor lapor ada error, langsung buang Kertas Buram! (Rollback)
		tx.Rollback()
		return err
	}

	// 4. Jika Mandor bilang semua aman, salin ke Buku Besar secara permanen! (Commit)
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

// GetTx adalah fungsi untuk Staf (Repository) mengecek Map Dokumennya.
// "Apakah ada Kertas Buram di map ini? Kalau ada, saya tulis di sana.
// Kalau tidak ada, saya tulis langsung di Buku Besar biasa (defaultDB)."
func GetTx(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx // Ketemu Kertas Buram!
	}
	return defaultDB // Tidak ada Kertas Buram, pakai Buku Besar.
}
