package usecase_test

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/domain/mocks"
	"github.com/faridlan/sikon-api/internal/usecase"
)

func TestUploadUsecase_UploadImage(t *testing.T) {
	mockStorage := new(mocks.StorageService)
	uc := usecase.NewUploadUsecase(mockStorage, time.Second*2)

	folder := "products"

	t.Run("Success", func(t *testing.T) {
		// Buat dummy file header
		file := &multipart.FileHeader{
			Filename: "test-image.jpg",
			Size:     1024, // 1KB
		}

		// Mock: Storage harus dipanggil dan berhasil
		mockStorage.On("UploadFile", mock.Anything, file, folder, mock.Anything).
			Return("https://storage.com/image.jpg", nil).Once()

		result, err := uc.UploadImage(context.Background(), file, folder)

		assert.NoError(t, err)
		assert.Equal(t, "https://storage.com/image.jpg", result)
		mockStorage.AssertExpectations(t)
	})

	t.Run("Error - File Too Large", func(t *testing.T) {
		file := &multipart.FileHeader{
			Filename: "big-image.jpg",
			Size:     3 * 1024 * 1024, // 3MB (Melebihi 2MB limit)
		}

		result, err := uc.UploadImage(context.Background(), file, folder)

		assert.Error(t, err)
		assert.Equal(t, "", result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
		assert.Equal(t, "Ukuran file maksimal 2MB", appErr.Message)
	})

	t.Run("Error - Invalid Extension", func(t *testing.T) {
		file := &multipart.FileHeader{
			Filename: "malicious.exe",
			Size:     1024,
		}

		result, err := uc.UploadImage(context.Background(), file, folder)

		assert.Error(t, err)
		assert.Equal(t, "", result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
		assert.Equal(t, "Format file harus JPG, PNG, atau WEBP", appErr.Message)
	})

	t.Run("Error - Storage Service Failure", func(t *testing.T) {
		file := &multipart.FileHeader{
			Filename: "test.png",
			Size:     1024,
		}

		// Mock: Storage gagal diupload
		mockStorage.On("UploadFile", mock.Anything, file, folder, mock.Anything).
			Return("", errors.New("supabase unavailable")).Once()

		result, err := uc.UploadImage(context.Background(), file, folder)

		assert.Error(t, err)
		assert.Equal(t, "", result)
		assert.Contains(t, err.Error(), "supabase unavailable")
		mockStorage.AssertExpectations(t)
	})
}
