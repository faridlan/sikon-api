package integration_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

func TestProduct_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)
	category := tests.SeedCategory(db, "Kaos Sablon")

	t.Run("Upload Image & Create Product - Success", func(t *testing.T) {
		// 1. UPLOAD IMAGE (Multipart)
		// Bagian ini tidak berubah karena endpoint upload tetap mengembalikan 1 URL
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Buat file dummy
		part, _ := writer.CreateFormFile("file", "test-image.jpg")
		part.Write([]byte("fake-image-content")) // Isi file palsu
		writer.Close()

		reqUpload := tests.AuthenticatedRequest("POST", "/api/uploads/image?folder=products", body, "test-user", "test-user@sikon.com", domain.RoleOwner)
		reqUpload.Header.Set("Content-Type", writer.FormDataContentType())

		respUpload, err := app.Test(reqUpload, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, respUpload.StatusCode)

		var uploadResp utils.SuccessResponse[map[string]string]
		json.NewDecoder(respUpload.Body).Decode(&uploadResp)
		imageURL := uploadResp.Data["url"]
		assert.NotEmpty(t, imageURL)

		// 2. CREATE PRODUCT (JSON)
		reqBody := dto.ProductCreateRequest{
			CategoryID:  category.ID,
			Name:        "Kaos Testing",
			Description: "Bahan Combed",
			BasePrice:   50000,
			ImageURLs:   []string{imageURL}, // 🚨 Masukkan URL hasil upload ke dalam array
		}
		bodyJson, _ := json.Marshal(reqBody)

		reqCreate := tests.AuthenticatedRequest("POST", "/api/products", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		reqCreate.Header.Set("Content-Type", "application/json")

		respCreate, err := app.Test(reqCreate, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, respCreate.StatusCode)

		var prodResp utils.SuccessResponse[dto.ProductResponse]
		json.NewDecoder(respCreate.Body).Decode(&prodResp)

		// 🚨 Verifikasi bahwa array Images terisi dengan benar
		assert.Equal(t, 1, len(prodResp.Data.Images))
		assert.Equal(t, imageURL, prodResp.Data.Images[0].ImageURL)
		assert.True(t, prodResp.Data.Images[0].IsPrimary) // Karena ini gambar pertama/satu-satunya
	})

	t.Run("Upload Image - Failed (Invalid Type)", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "malicious.exe") // Extension salah
		part.Write([]byte("fake-content"))
		writer.Close()

		req := tests.AuthenticatedRequest("POST", "/api/uploads/image", body, "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}
