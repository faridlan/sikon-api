package integration_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

func TestProduct_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)
	category := tests.SeedCategory(db, "Kaos Sablon")

	t.Run("Upload Image & Create Product - Success", func(t *testing.T) {
		// 1. UPLOAD IMAGE (Multipart)
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Buat file dummy
		part, _ := writer.CreateFormFile("file", "test-image.jpg")
		part.Write([]byte("fake-image-content")) // Isi file palsu
		writer.Close()

		reqUpload := httptest.NewRequest("POST", "/api/uploads/image?folder=products", body)
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
			ImageURL:    imageURL,
		}
		bodyJson, _ := json.Marshal(reqBody)

		reqCreate := httptest.NewRequest("POST", "/api/products", bytes.NewBuffer(bodyJson))
		reqCreate.Header.Set("Content-Type", "application/json")

		respCreate, err := app.Test(reqCreate, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, respCreate.StatusCode)

		var prodResp utils.SuccessResponse[dto.ProductResponse]
		json.NewDecoder(respCreate.Body).Decode(&prodResp)

		assert.Equal(t, imageURL, prodResp.Data.ImageURL)
	})

	t.Run("Upload Image - Failed (Invalid Type)", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "malicious.exe") // Extension salah
		part.Write([]byte("fake-content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/api/uploads/image", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}
