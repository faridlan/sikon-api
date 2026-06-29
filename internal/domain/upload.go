package domain

import (
	"context"
	"mime/multipart"
)

// StorageService adalah interface untuk infrastruktur pihak ketiga (Supabase/S3)
type StorageService interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader, folder string, fileName string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
}

// UploadUsecase adalah interface untuk logika bisnis validasi file
type UploadUsecase interface {
	UploadImage(ctx context.Context, file *multipart.FileHeader, folder string) (string, error)
}
