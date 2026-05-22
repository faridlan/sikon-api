package domain

import "errors"

var (
	ErrInternalServerError = errors.New("terjadi kesalahan pada server")
	ErrNotFound            = errors.New("data tidak ditemukan")
	ErrConflict            = errors.New("data sudah ada (konflik)")
	ErrBadParamInput       = errors.New("parameter atau format data tidak valid")
	ErrLimitExceeded       = errors.New("kuota API eksternal habis")
	ErrUnauthorized        = errors.New("akses tidak sah, token invalid atau expired")
)

// 2. CUSTOM ERROR STRUCT
type AppError struct {
	ErrType error  // Menyimpan kategori (misal: ErrNotFound)
	Message string // Menyimpan pesan spesifik (misal: "Buku tidak ditemukan")
}

// 3. IMPLEMENTASI INTERFACE ERROR
// Ini yang akan dipanggil oleh err.Error() di HandleDomainError
func (e *AppError) Error() string {
	return e.Message
}

// 4. IMPLEMENTASI UNWRAP (SANGAT KRUSIAL)
// Ini membuat fungsi errors.Is() milik Golang tetap bisa membaca kategori aslinya
func (e *AppError) Unwrap() error {
	return e.ErrType
}

// 5. HELPER FUNCTION (Agar mudah dipanggil di Usecase)
func NewError(errType error, message string) error {
	return &AppError{
		ErrType: errType,
		Message: message,
	}
}
