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

func StringPtr(s string) *string {
	return &s
}

func setupTxMock(mockTx *mocks.TransactionManager) {
	mockTx.On("RunInTransaction", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		}).
		Return(nil)
}

func TestProductUsecase_CreateProduct(t *testing.T) {
	mockProductRepo := new(mocks.ProductRepository)
	mockCategoryRepo := new(mocks.CategoryRepository)
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

	input := domain.ProductCreateInput{
		CategoryID:  "cat-123",
		Name:        "Kemeja Taktikal Premium 7200",
		Description: "Bahan Ripstop Anti Robek",
		BasePrice:   185000,
	}

	t.Run("Success", func(t *testing.T) {
		setupTxMock(mockTxManager)

		mockCategoryRepo.On("GetByID", mock.Anything, input.CategoryID).
			Return(&domain.Category{ID: input.CategoryID}, nil).Once()

		mockProductRepo.On("GetBySlug", mock.Anything, "kemeja-taktikal-premium-7200").
			Return(nil, domain.ErrNotFound).Once()

		mockProductRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return p.Name == input.Name && p.CategoryID == input.CategoryID && p.Slug == "kemeja-taktikal-premium-7200"
		})).Return(nil).Once()

		result, err := uc.CreateProduct(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		assert.Equal(t, "kemeja-taktikal-premium-7200", result.Slug)
		mockCategoryRepo.AssertExpectations(t)
		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Success - With Fabrics, Wholesale and DesignModel", func(t *testing.T) {
		setupTxMock(mockTxManager)

		inputFull := domain.ProductCreateInput{
			CategoryID:    "cat-123",
			Name:          "Kemeja Taktikal Premium 7200",
			BasePrice:     185000,
			GSMInfo:       "210gsm",
			FabricSummary: "Ripstop",
			KeyFeatures:   []string{"Bahan anti robek", "Dual pocket"},
			ImageURLs:     []string{"https://example.com/1.jpg"},
			Fabrics: []domain.ProductFabricInput{
				{
					Name:      "Ripstop Cotton",
					BasePrice: 185000,
					IsDefault: true,
					Colors: []domain.FabricColorInput{
						{Name: "Olive", HexCode: "#4b5320"},
					},
				},
			},
			Wholesale: []domain.WholesalePriceInput{
				{MinQty: 6, UnitPrice: 175000},
			},
			DesignModel: &domain.ProductModelInput{
				Name: "Series 1 — Long Sleeve",
				Type: "long_sleeve",
				Views: []domain.ProductModelViewInput{
					{Side: "front", ArtURL: "https://example.com/front-art.png", MaskURL: "https://example.com/front-mask.png"},
				},
			},
		}

		mockCategoryRepo.On("GetByID", mock.Anything, inputFull.CategoryID).
			Return(&domain.Category{ID: inputFull.CategoryID}, nil).Once()

		mockProductRepo.On("GetBySlug", mock.Anything, "kemeja-taktikal-premium-7200").
			Return(nil, domain.ErrNotFound).Once()

		mockProductRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return len(p.Fabrics) == 1 && len(p.Wholesale) == 1 && p.DesignModel != nil
		})).Return(nil).Once()

		result, err := uc.CreateProduct(context.Background(), inputFull)

		assert.NoError(t, err)
		assert.Len(t, result.Fabrics, 1)
		assert.Len(t, result.Wholesale, 1)
		assert.NotNil(t, result.DesignModel)
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
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

	mockID := "prod-123"
	mockProduct := &domain.Product{
		ID:         mockID,
		CategoryID: "cat-123",
		Name:       "Kemeja Taktikal Premium 7200",
		Slug:       "kemeja-taktikal-premium-7200",
	}

	t.Run("Success By ID", func(t *testing.T) {
		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(mockProduct, nil).Once()

		result, err := uc.GetProduct(context.Background(), mockID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, mockProduct.Name, result.Name)

		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Error - ID Not Found", func(t *testing.T) {
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

func TestProductUsecase_GetProductBySlug(t *testing.T) {
	mockProductRepo := new(mocks.ProductRepository)
	mockCategoryRepo := new(mocks.CategoryRepository)
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

	mockSlug := "kemeja-taktikal-premium-7200"
	mockProduct := &domain.Product{
		ID:         "prod-123",
		CategoryID: "cat-123",
		Name:       "Kemeja Taktikal Premium 7200",
		Slug:       mockSlug,
	}

	t.Run("Success By Slug", func(t *testing.T) {
		mockProductRepo.On("GetBySlug", mock.Anything, mockSlug).Return(mockProduct, nil).Once()

		result, err := uc.GetProductBySlug(context.Background(), mockSlug)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, mockSlug, result.Slug)

		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Error - Slug Not Found", func(t *testing.T) {
		mockProductRepo.On("GetBySlug", mock.Anything, mockSlug).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetProductBySlug(context.Background(), mockSlug)

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
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

	query := domain.PaginationQuery{Page: 2, Limit: 5}
	filter := domain.ProductFilter{
		Search:     "Kemeja",
		CategoryID: "cat-1",
		MinPrice:   100000,
		MaxPrice:   200000,
		SortBy:     "popular",
	}

	mockProducts := []domain.Product{
		{ID: "prod-1", Name: "Kemeja Lengan Pendek"},
		{ID: "prod-2", Name: "Kemeja Lengan Panjang"},
	}
	var totalItems int64 = 12

	t.Run("Success", func(t *testing.T) {
		mockProductRepo.On("Fetch", mock.Anything, filter, 5, 5).
			Return(mockProducts, totalItems, nil).Once()

		products, meta, err := uc.ListProducts(context.Background(), filter, query)

		assert.NoError(t, err)
		assert.Len(t, products, 2)
		assert.Equal(t, int64(12), meta.TotalItems)
		assert.Equal(t, 3, meta.TotalPages)
		assert.Equal(t, 2, meta.CurrentPage)

		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Error_From_Repository", func(t *testing.T) {
		expectedErr := errors.New("database connection failed")

		mockProductRepo.On("Fetch", mock.Anything, filter, 5, 5).
			Return(nil, int64(0), expectedErr).Once()

		products, meta, err := uc.ListProducts(context.Background(), filter, query)

		assert.Error(t, err)
		assert.Nil(t, products)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 0, meta.TotalPages)

		mockProductRepo.AssertExpectations(t)
	})
}

func TestProductUsecase_UpdateProduct(t *testing.T) {
	mockID := "prod-123"

	t.Run("Success - Update without Category Change (No Image Change)", func(t *testing.T) {
		mockProductRepo := new(mocks.ProductRepository)
		mockCategoryRepo := new(mocks.CategoryRepository)
		mockStorageService := new(mocks.StorageService)
		mockTxManager := new(mocks.TransactionManager)

		uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

		existingProd := &domain.Product{
			ID:         mockID,
			CategoryID: "cat-old",
			Name:       "Kemeja Lama",
			BasePrice:  150000,
		}

		input := domain.ProductUpdateInput{
			Name:      "Kemeja Baru",
			BasePrice: 185000,
		}

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd, nil).Once()

		setupTxMock(mockTxManager)

		mockProductRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return p.Name == "Kemeja Baru" && p.BasePrice == 185000 && p.CategoryID == "cat-old"
		})).Return(nil).Once()

		result, err := uc.UpdateProduct(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "Kemeja Baru", result.Name)

		mockProductRepo.AssertExpectations(t)
		mockCategoryRepo.AssertExpectations(t)
		mockStorageService.AssertExpectations(t)
	})

	t.Run("Success - Update with Image Change (Triggers Background Delete)", func(t *testing.T) {
		mockProductRepo := new(mocks.ProductRepository)
		mockCategoryRepo := new(mocks.CategoryRepository)
		mockStorageService := new(mocks.StorageService)
		mockTxManager := new(mocks.TransactionManager)

		uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

		existingProd := &domain.Product{
			ID:         mockID,
			CategoryID: "cat-old",
			Images:     []domain.ProductImage{{ImageURL: "https://example.com/old.jpg"}},
		}

		input := domain.ProductUpdateInput{
			ImageURLs: []string{"https://example.com/new.jpg"},
		}

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd, nil).Once()

		setupTxMock(mockTxManager)
		mockProductRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Once()

		mockStorageService.On("DeleteFile", mock.Anything, "https://example.com/old.jpg").
			Return(nil).Once()

		_, err := uc.UpdateProduct(context.Background(), mockID, input)

		assert.NoError(t, err)
		time.Sleep(20 * time.Millisecond)

		mockProductRepo.AssertExpectations(t)
		mockStorageService.AssertExpectations(t)
	})

	t.Run("Success - Update with Category Change", func(t *testing.T) {
		mockProductRepo := new(mocks.ProductRepository)
		mockCategoryRepo := new(mocks.CategoryRepository)
		mockStorageService := new(mocks.StorageService)
		mockTxManager := new(mocks.TransactionManager)

		uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

		existingProd := &domain.Product{ID: mockID, CategoryID: "cat-old", Name: "Kemeja Lama"}
		input := domain.ProductUpdateInput{CategoryID: "cat-new", Name: "Kemeja Premium"}

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd, nil).Once()

		mockCategoryRepo.On("GetByID", mock.Anything, "cat-new").
			Return(&domain.Category{ID: "cat-new"}, nil).Once()

		setupTxMock(mockTxManager)
		mockProductRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
			return p.CategoryID == "cat-new" && p.Name == "Kemeja Premium"
		})).Return(nil).Once()

		result, err := uc.UpdateProduct(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "cat-new", result.CategoryID)

		mockProductRepo.AssertExpectations(t)
		mockCategoryRepo.AssertExpectations(t)
	})
}

func TestProductUsecase_DeleteProduct(t *testing.T) {
	mockID := "prod-123"

	t.Run("Success - Delete Product without Images", func(t *testing.T) {
		mockProductRepo := new(mocks.ProductRepository)
		mockCategoryRepo := new(mocks.CategoryRepository)
		mockStorageService := new(mocks.StorageService)
		mockTxManager := new(mocks.TransactionManager)

		uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

		existingProd := &domain.Product{ID: mockID, Name: "Kemeja Lama", Images: []domain.ProductImage{}}

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd, nil).Once()
		mockProductRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()

		err := uc.DeleteProduct(context.Background(), mockID)

		assert.NoError(t, err)
		mockProductRepo.AssertExpectations(t)
		mockStorageService.AssertExpectations(t)
	})

	t.Run("Success - Delete Product with Images and Canvas Models", func(t *testing.T) {
		mockProductRepo := new(mocks.ProductRepository)
		mockCategoryRepo := new(mocks.CategoryRepository)
		mockStorageService := new(mocks.StorageService)
		mockTxManager := new(mocks.TransactionManager)

		uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

		existingProd := &domain.Product{
			ID:   mockID,
			Name: "Kemeja Taktikal",
			Images: []domain.ProductImage{
				{ImageURL: "https://example.com/img1.jpg"},
			},
			DesignModel: &domain.ProductModel{
				Views: []domain.ProductModelView{
					{ArtURL: "https://example.com/art.png", MaskURL: "https://example.com/mask.png"},
				},
			},
		}

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd, nil).Once()
		mockProductRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()

		mockStorageService.On("DeleteFile", mock.Anything, "https://example.com/img1.jpg").Return(nil).Once()
		mockStorageService.On("DeleteFile", mock.Anything, "https://example.com/art.png").Return(nil).Once()
		mockStorageService.On("DeleteFile", mock.Anything, "https://example.com/mask.png").Return(nil).Once()

		err := uc.DeleteProduct(context.Background(), mockID)

		assert.NoError(t, err)
		time.Sleep(20 * time.Millisecond)

		mockProductRepo.AssertExpectations(t)
		mockStorageService.AssertExpectations(t)
	})

	t.Run("Error - Failed to Delete DB", func(t *testing.T) {
		mockProductRepo := new(mocks.ProductRepository)
		mockCategoryRepo := new(mocks.CategoryRepository)
		mockStorageService := new(mocks.StorageService)
		mockTxManager := new(mocks.TransactionManager)

		uc := usecase.NewProductUsecase(mockProductRepo, mockCategoryRepo, mockStorageService, mockTxManager, time.Second*2)

		existingProd := &domain.Product{
			ID:     mockID,
			Images: []domain.ProductImage{{ImageURL: "https://example.com/delete-me.jpg"}},
		}
		dbErr := errors.New("db connection failed")

		mockProductRepo.On("GetByID", mock.Anything, mockID).Return(existingProd, nil).Once()
		mockProductRepo.On("Delete", mock.Anything, mockID).Return(dbErr).Once()

		err := uc.DeleteProduct(context.Background(), mockID)

		assert.Error(t, err)
		assert.Equal(t, dbErr, err)

		mockProductRepo.AssertExpectations(t)
		mockStorageService.AssertExpectations(t)
	})
}
