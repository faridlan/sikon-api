package usecase

import (
	"context"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/faridlan/sikon-api/internal/domain"
)

type uploadUsecase struct {
	storageService domain.StorageService
	contextTimeout time.Duration
}

func NewUploadUsecase(ss domain.StorageService, timeout time.Duration) domain.UploadUsecase {
	return &uploadUsecase{
		storageService: ss,
		contextTimeout: timeout,
	}
}

func (u *uploadUsecase) UploadImage(c context.Context, file *multipart.FileHeader, folder string) (string, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Validasi Ukuran (Maksimal 2MB)
	if file.Size > 2*1024*1024 {
		return "", domain.NewError(domain.ErrBadParamInput, "Ukuran file maksimal 2MB")
	}

	// 2. Validasi Ekstensi
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", domain.NewError(domain.ErrBadParamInput, "Format file harus JPG, PNG, atau WEBP")
	}

	// 3. Generate Nama Unik
	fileName := uuid.New().String()

	// 4. Proses Upload via Infrastructure
	return u.storageService.UploadFile(ctx, file, folder, fileName)
}
