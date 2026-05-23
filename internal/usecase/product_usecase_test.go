package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/domain/mocks"
	"github.com/faridlan/sikon-api/internal/usecase"
)

func TestProductUsecase_CreateProduct(t *testing.T) {
	mockProductRepo := new(mocks.ProductRepository)
	mockCategoryRepo := new(mocks.CategoryRepository)
	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, time.Second*2)

	input := domain.ProductCreateInput{
		CategoryID: "cat-123",
		Name:       "Kaos Cotton",
		BasePrice:  50000,
	}

	t.Run("Success", func(t *testing.T) {
		// Mock: Kategori harus ditemukan
		mockCategoryRepo.On("GetByID", mock.Anything, input.CategoryID).
			Return(&domain.Category{ID: input.CategoryID}, nil).Once()

		// Mock: Produk berhasil dibuat
		mockProductRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return p.Name == input.Name && p.CategoryID == input.CategoryID
		})).Return(nil).Once()

		result, err := uc.CreateProduct(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockCategoryRepo.AssertExpectations(t)
		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Error - Category Not Found", func(t *testing.T) {
		mockCategoryRepo.On("GetByID", mock.Anything, input.CategoryID).
			Return(nil, domain.ErrNotFound).Once()

		result, err := uc.CreateProduct(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockCategoryRepo.AssertExpectations(t)
	})
}

func TestProductUsecase_UpdateProduct(t *testing.T) {
	mockProductRepo := new(mocks.ProductRepository)
	mockCategoryRepo := new(mocks.CategoryRepository)
	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, time.Second*2)

	mockID := "prod-123"
	existingProd := &domain.Product{ID: mockID, CategoryID: "cat-old", Name: "Lama"}

	t.Run("Success - Update with Category Change", func(t *testing.T) {
		input := domain.ProductUpdateInput{CategoryID: "cat-new", Name: "Baru"}

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd, nil).Once()

		// Kategori baru harus divalidasi
		mockCategoryRepo.On("GetByID", mock.Anything, "cat-new").
			Return(&domain.Category{ID: "cat-new"}, nil).Once()

		mockProductRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return p.CategoryID == "cat-new" && p.Name == "Baru"
		})).Return(nil).Once()

		result, err := uc.UpdateProduct(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "Baru", result.Name)
		assert.Equal(t, "cat-new", result.CategoryID)
		mockCategoryRepo.AssertExpectations(t)
		mockProductRepo.AssertExpectations(t)
	})
}
