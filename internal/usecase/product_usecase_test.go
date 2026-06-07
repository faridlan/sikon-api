package usecase_test

import (
	"context"
	"errors"
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
		CategoryID:  "cat-123",
		Name:        "Kaos Cotton",
		Description: "Bahan Halus",
		BasePrice:   50000,
	}

	t.Run("Success", func(t *testing.T) {
		// Mock: Kategori harus ditemukan
		mockCategoryRepo.On("GetByID", mock.Anything, input.CategoryID).
			Return(&domain.Category{ID: input.CategoryID}, nil).Once()

		// Mock: Produk berhasil dibuat
		mockProductRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return p.Name == input.Name && p.CategoryID == input.CategoryID && p.BasePrice == input.BasePrice
		})).Return(nil).Once()

		result, err := uc.CreateProduct(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		mockCategoryRepo.AssertExpectations(t)
		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Error - Category Not Found", func(t *testing.T) {
		mockCategoryRepo.On("GetByID", mock.Anything, input.CategoryID).
			Return(nil, domain.ErrNotFound).Once()

		result, err := uc.CreateProduct(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)

		mockCategoryRepo.AssertExpectations(t)
	})
}

func TestProductUsecase_GetProduct(t *testing.T) {
	mockProductRepo := new(mocks.ProductRepository)
	mockCategoryRepo := new(mocks.CategoryRepository)
	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, time.Second*2)

	mockID := "prod-123"
	mockProduct := &domain.Product{
		ID:         mockID,
		CategoryID: "cat-123",
		Name:       "Kaos Cotton",
	}

	t.Run("Success", func(t *testing.T) {
		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(mockProduct, nil).Once()

		result, err := uc.GetProduct(context.Background(), mockID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, mockProduct.Name, result.Name)

		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetProduct(context.Background(), mockID)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrNotFound, appErr.ErrType)

		mockProductRepo.AssertExpectations(t)
	})
}

func TestProductUsecase_ListProducts(t *testing.T) {
	mockProductRepo := new(mocks.ProductRepository)
	mockCategoryRepo := new(mocks.CategoryRepository)
	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, time.Second*2)

	// 1. Siapkan mock input query dan filter
	query := domain.PaginationQuery{Page: 2, Limit: 5}
	filter := domain.ProductFilter{
		Search:     "Kemeja",
		CategoryID: "cat-1",
	}

	mockProducts := []domain.Product{
		{ID: "prod-1", Name: "Kemeja Lengan Pendek"},
		{ID: "prod-2", Name: "Kemeja Lengan Panjang"},
	}
	var totalItems int64 = 12

	t.Run("Success", func(t *testing.T) {
		// 2. Tambahkan parameter 'filter' pada argumen mock.On()
		// Offset dihitung: (Page 2 - 1) * 5 Limit = 5
		mockProductRepo.On("Fetch", mock.Anything, filter, 5, 5).
			Return(mockProducts, totalItems, nil).Once()

		// 3. Sisipkan parameter 'filter' saat memanggil fungsi usecase
		products, meta, err := uc.ListProducts(context.Background(), filter, query)

		assert.NoError(t, err)
		assert.Len(t, products, 2)
		assert.Equal(t, int64(12), meta.TotalItems)
		// Total pages: ceil(12 / 5) = 3
		assert.Equal(t, 3, meta.TotalPages)
		assert.Equal(t, 2, meta.CurrentPage)

		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Error_From_Repository", func(t *testing.T) {
		// Skenario jika database gagal/error saat mencari data dengan filter
		expectedErr := errors.New("database connection failed")

		mockProductRepo.On("Fetch", mock.Anything, filter, 5, 5).
			Return(nil, int64(0), expectedErr).Once()

		products, meta, err := uc.ListProducts(context.Background(), filter, query)

		// Verifikasi error handling
		assert.Error(t, err)
		assert.Nil(t, products)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 0, meta.TotalPages) // Meta harus kosong saat error

		mockProductRepo.AssertExpectations(t)
	})
}

func TestProductUsecase_UpdateProduct(t *testing.T) {
	mockProductRepo := new(mocks.ProductRepository)
	mockCategoryRepo := new(mocks.CategoryRepository)
	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, time.Second*2)

	mockID := "prod-123"
	existingProd := &domain.Product{
		ID:          mockID,
		CategoryID:  "cat-old",
		Name:        "Kaos Lama",
		Description: "Bahan Biasa",
		BasePrice:   30000,
	}

	t.Run("Success - Update without Category Change", func(t *testing.T) {
		input := domain.ProductUpdateInput{
			Name:      "Kaos Baru",
			BasePrice: 40000,
		}

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd, nil).Once()

		// Karena CategoryID tidak dikirim ("" atau tidak berubah), CategoryRepo JANGAN dipanggil
		mockProductRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return p.Name == "Kaos Baru" && p.BasePrice == 40000 && p.CategoryID == "cat-old" // Kategori tetap
		})).Return(nil).Once()

		result, err := uc.UpdateProduct(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "Kaos Baru", result.Name)
		assert.Equal(t, float64(40000), result.BasePrice)

		mockProductRepo.AssertExpectations(t)
		mockCategoryRepo.AssertExpectations(t) // Akan pass jika tidak dipanggil sama sekali
	})

	t.Run("Success - Update with Category Change", func(t *testing.T) {
		input := domain.ProductUpdateInput{
			CategoryID: "cat-new",
			Name:       "Kaos Premium",
		}

		// Reset state objek lama untuk tes ini
		existingProd2 := &domain.Product{ID: mockID, CategoryID: "cat-old", Name: "Kaos Lama"}

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd2, nil).Once()

		// Kategori baru harus divalidasi
		mockCategoryRepo.On("GetByID", mock.Anything, "cat-new").
			Return(&domain.Category{ID: "cat-new"}, nil).Once()

		mockProductRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return p.CategoryID == "cat-new" && p.Name == "Kaos Premium"
		})).Return(nil).Once()

		result, err := uc.UpdateProduct(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "Kaos Premium", result.Name)
		assert.Equal(t, "cat-new", result.CategoryID)

		mockProductRepo.AssertExpectations(t)
		mockCategoryRepo.AssertExpectations(t)
	})

	t.Run("Error - New Category Not Found", func(t *testing.T) {
		input := domain.ProductUpdateInput{CategoryID: "cat-invalid"}
		existingProd3 := &domain.Product{ID: mockID, CategoryID: "cat-old"}

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd3, nil).Once()

		// Kategori baru dicari tapi tidak ketemu
		mockCategoryRepo.On("GetByID", mock.Anything, "cat-invalid").
			Return(nil, domain.ErrNotFound).Once()

		result, err := uc.UpdateProduct(context.Background(), mockID, input)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)

		mockProductRepo.AssertExpectations(t)
		mockCategoryRepo.AssertExpectations(t)
	})
}

func TestProductUsecase_DeleteProduct(t *testing.T) {
	mockProductRepo := new(mocks.ProductRepository)
	mockCategoryRepo := new(mocks.CategoryRepository)
	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, time.Second*2)

	mockID := "prod-123"
	existingProd2 := &domain.Product{ID: mockID, CategoryID: "cat-old", Name: "Kaos Lama"}

	t.Run("Success", func(t *testing.T) {

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd2, nil).Once()
		mockProductRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()

		err := uc.DeleteProduct(context.Background(), mockID)

		assert.NoError(t, err)
		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Error - Failed to Delete", func(t *testing.T) {
		dbErr := errors.New("db connection failed")

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd2, nil).Once()
		mockProductRepo.On("Delete", mock.Anything, mockID).Return(dbErr).Once()

		err := uc.DeleteProduct(context.Background(), mockID)

		assert.Error(t, err)
		assert.Equal(t, dbErr, err)
		mockProductRepo.AssertExpectations(t)
	})
}
