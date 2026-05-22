package postgres

import (
	"errors"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

// TranslateError mengubah error bawaan GORM menjadi error standar Domain kita
func TranslateError(err error) error {
	if err == nil {
		return nil
	}

	// Gunakan errors.Is (lebih aman daripada == untuk error di Golang modern)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}

	return err
}
