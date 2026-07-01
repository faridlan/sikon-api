package supabase

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/faridlan/sikon-api/internal/domain"
)

type supabaseStorage struct {
	projectURL string
	apiKey     string
	bucketName string
}

func NewSupabaseStorage(url, key, bucket string) domain.StorageService {
	return &supabaseStorage{
		projectURL: url,
		apiKey:     key,
		bucketName: bucket,
	}
}

func (s *supabaseStorage) UploadFile(ctx context.Context, fileHeader *multipart.FileHeader, folder string, fileName string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	ext := filepath.Ext(fileHeader.Filename)
	fullPath := fmt.Sprintf("%s/%s%s", folder, fileName, ext)
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.projectURL, s.bucketName, fullPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, file)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", fileHeader.Header.Get("Content-Type"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gagal upload ke supabase: %s", string(bodyBytes))
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.projectURL, s.bucketName, fullPath)
	return publicURL, nil
}

func (s *supabaseStorage) DeleteFile(ctx context.Context, fileURL string) error {
	// 1. Ekstrak jalur relatif file dari URL Publik
	// Kita buat prefix URL publik yang sama persis seperti saat kita meng-generate-nya di UploadFile
	prefix := fmt.Sprintf("%s/storage/v1/object/public/%s/", s.projectURL, s.bucketName)

	// Pastikan URL yang dikirim benar-benar berasal dari bucket kita
	if !strings.HasPrefix(fileURL, prefix) {
		return fmt.Errorf("URL file tidak valid atau bukan berasal dari bucket yang tepat")
	}

	// Potong prefix-nya, sisakan hanya "folder/filename.ext"
	fullPath := strings.TrimPrefix(fileURL, prefix)

	// 2. Siapkan Endpoint DELETE Supabase
	// Endpoint untuk delete object: DELETE /storage/v1/object/{bucketName}/{fullPath}
	deleteURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.projectURL, s.bucketName, fullPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return err
	}

	// 3. Set Header Otorisasi (Gunakan Service Role Key yang ada di s.apiKey)
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	// 4. Eksekusi Request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 5. Cek Status Code
	// Supabase biasanya mengembalikan 200 OK jika berhasil dihapus
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gagal menghapus file di supabase: %s", string(bodyBytes))
	}

	return nil
}
