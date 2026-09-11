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

func setupMaterialTest() (*mocks.MaterialRepository, domain.MaterialUsecase) {
	mockRepo := new(mocks.MaterialRepository)
	uc := usecase.NewMaterialUsecase(mockRepo, 2*time.Second)
	return mockRepo, uc
}

// ==========================================
// TESTS: MATERIAL
// ==========================================

func TestMaterialUsecase_CreateMaterial(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupMaterialTest()
		input := domain.MaterialCreateInput{
			Name:      "Kain Ripstop",
			Unit:      "meter",
			UnitPrice: 25000,
			Category:  "kain",
		}

		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(m *domain.Material) bool {
			return m.Name == "Kain Ripstop" && m.UnitPrice == 25000
		})).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*domain.Material)
			arg.ID = "mat-123" // Simulasikan DB memberi ID
		}).Once()

		result, err := uc.CreateMaterial(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "mat-123", result.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed - Validation Errors", func(t *testing.T) {
		_, uc := setupMaterialTest()

		tests := []struct {
			name  string
			input domain.MaterialCreateInput
		}{
			{"Empty Name", domain.MaterialCreateInput{Name: "", Unit: "meter", UnitPrice: 1000}},
			{"Empty Unit", domain.MaterialCreateInput{Name: "Kain", Unit: "", UnitPrice: 1000}},
			{"Zero Price", domain.MaterialCreateInput{Name: "Kain", Unit: "meter", UnitPrice: 0}},
			{"Negative Price", domain.MaterialCreateInput{Name: "Kain", Unit: "meter", UnitPrice: -100}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.CreateMaterial(context.Background(), tc.input)
				assert.Error(t, err)
				assert.Nil(t, result)

				var appErr *domain.AppError
				assert.True(t, errors.As(err, &appErr))
				assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
			})
		}
	})

	t.Run("Failed - Repo Error", func(t *testing.T) {
		mockRepo, uc := setupMaterialTest()
		input := domain.MaterialCreateInput{Name: "Kain", Unit: "meter", UnitPrice: 1000}

		expectedErr := errors.New("db error")
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(expectedErr).Once()

		result, err := uc.CreateMaterial(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestMaterialUsecase_GetMaterial(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupMaterialTest()
		id := "mat-123"
		expected := &domain.Material{ID: id, Name: "Kain Ripstop"}

		mockRepo.On("GetByID", mock.Anything, id).Return(expected, nil).Once()

		result, err := uc.GetMaterial(context.Background(), id)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed - Empty ID", func(t *testing.T) {
		_, uc := setupMaterialTest()

		result, err := uc.GetMaterial(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Failed - Not Found", func(t *testing.T) {
		mockRepo, uc := setupMaterialTest()
		mockRepo.On("GetByID", mock.Anything, "mat-404").Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetMaterial(context.Background(), "mat-404")

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestMaterialUsecase_ListMaterials(t *testing.T) {
	t.Run("Success - Default Pagination", func(t *testing.T) {
		mockRepo, uc := setupMaterialTest()
		query := domain.PaginationQuery{Page: 1, Limit: 10}
		expectedData := []domain.Material{{ID: "1"}, {ID: "2"}}

		// Offset (1-1)*10 = 0
		mockRepo.On("Fetch", mock.Anything, 10, 0).Return(expectedData, int64(15), nil).Once()

		result, meta, err := uc.ListMaterials(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(15), meta.TotalItems)
		assert.Equal(t, 2, meta.TotalPages) // ceil(15/10) = 2
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Limit Dinormalisasi Kalau Invalid", func(t *testing.T) {
		mockRepo, uc := setupMaterialTest()
		query := domain.PaginationQuery{Page: 0, Limit: 0} // harus jadi Page=1, Limit=10

		mockRepo.On("Fetch", mock.Anything, 10, 0).Return([]domain.Material{}, int64(0), nil).Once()

		_, meta, err := uc.ListMaterials(context.Background(), query)

		assert.NoError(t, err)
		assert.Equal(t, 1, meta.CurrentPage)
		assert.Equal(t, 10, meta.Limit)
		mockRepo.AssertExpectations(t)
	})
}

func TestMaterialUsecase_UpdateMaterial(t *testing.T) {
	t.Run("Success - Update Sebagian Field Saja", func(t *testing.T) {
		mockRepo, uc := setupMaterialTest()
		id := "mat-123"
		existing := &domain.Material{ID: id, Name: "Kain Lama", Unit: "meter", UnitPrice: 20000, Category: "kain"}

		mockRepo.On("GetByID", mock.Anything, id).Return(existing, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *domain.Material) bool {
			// Cuma harga yang berubah, field lain tetap
			return m.UnitPrice == 27000 && m.Name == "Kain Lama"
		})).Return(nil).Once()

		result, err := uc.UpdateMaterial(context.Background(), id, domain.MaterialUpdateInput{UnitPrice: 27000})

		assert.NoError(t, err)
		assert.Equal(t, float64(27000), result.UnitPrice)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed - Material Tidak Ditemukan", func(t *testing.T) {
		mockRepo, uc := setupMaterialTest()
		mockRepo.On("GetByID", mock.Anything, "mat-404").Return(nil, domain.ErrNotFound).Once()

		result, err := uc.UpdateMaterial(context.Background(), "mat-404", domain.MaterialUpdateInput{UnitPrice: 1000})

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestMaterialUsecase_DeleteMaterial(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupMaterialTest()
		id := "mat-123"

		mockRepo.On("Delete", mock.Anything, id).Return(nil).Once()

		err := uc.DeleteMaterial(context.Background(), id)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed - Empty ID", func(t *testing.T) {
		_, uc := setupMaterialTest()

		err := uc.DeleteMaterial(context.Background(), "")

		assert.Error(t, err)
	})
}

// ==========================================
// TESTS: PRODUCT MATERIAL (Resep / BOM)
// ==========================================

func setupProductMaterialTest() (*mocks.ProductMaterialRepository, *mocks.ProductRepository, *mocks.TransactionManager, domain.ProductMaterialUsecase) {
	mockPMRepo := new(mocks.ProductMaterialRepository)
	mockProductRepo := new(mocks.ProductRepository)
	mockTx := new(mocks.TransactionManager)

	// TransactionManager di-mock supaya benar-benar MENJALANKAN fungsi yang dibungkus,
	// bukan cuma pura-pura sukses tanpa eksekusi apa pun.
	mockTx.On("RunInTransaction", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(args.Get(0).(context.Context))
		}).
		Return(nil)

	uc := usecase.NewProductMaterialUsecase(mockPMRepo, mockProductRepo, nil, mockTx, 2*time.Second)
	return mockPMRepo, mockProductRepo, mockTx, uc
}

func TestProductMaterialUsecase_SetProductMaterials(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockPMRepo, mockProductRepo, _, uc := setupProductMaterialTest()
		productID := "prod-123"

		input := domain.SetProductMaterialsInput{
			ProductID: productID,
			Items: []domain.ProductMaterialItemInput{
				{MaterialID: "mat-1", QtyPerUnit: 1.2},
				{MaterialID: "mat-2", QtyPerUnit: 3},
			},
		}

		mockProductRepo.On("GetByID", mock.Anything, productID).Return(&domain.Product{ID: productID}, nil).Once()

		mockPMRepo.On("ReplaceForProduct", mock.Anything, productID, mock.MatchedBy(func(items []domain.ProductMaterial) bool {
			return len(items) == 2 && items[0].MaterialID == "mat-1" && items[1].QtyPerUnit == 3
		})).Return(nil).Once()

		expectedResult := []domain.ProductMaterial{
			{ID: "pm-1", ProductID: productID, MaterialID: "mat-1", QtyPerUnit: 1.2},
			{ID: "pm-2", ProductID: productID, MaterialID: "mat-2", QtyPerUnit: 3},
		}
		mockPMRepo.On("FetchByProduct", mock.Anything, productID).Return(expectedResult, nil).Once()

		result, err := uc.SetProductMaterials(context.Background(), input)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		mockProductRepo.AssertExpectations(t)
		mockPMRepo.AssertExpectations(t)
	})

	t.Run("Failed - Produk Tidak Ditemukan", func(t *testing.T) {
		_, mockProductRepo, _, uc := setupProductMaterialTest()
		productID := "prod-404"

		mockProductRepo.On("GetByID", mock.Anything, productID).Return(nil, domain.ErrNotFound).Once()

		input := domain.SetProductMaterialsInput{
			ProductID: productID,
			Items:     []domain.ProductMaterialItemInput{{MaterialID: "mat-1", QtyPerUnit: 1}},
		}

		result, err := uc.SetProductMaterials(context.Background(), input)

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Nil(t, result)
		mockProductRepo.AssertExpectations(t)
	})

	t.Run("Failed - Validation Errors", func(t *testing.T) {
		_, mockProductRepo, _, uc := setupProductMaterialTest()
		productID := "prod-123"
		mockProductRepo.On("GetByID", mock.Anything, productID).Return(&domain.Product{ID: productID}, nil)

		tests := []struct {
			name  string
			input domain.SetProductMaterialsInput
		}{
			{"Product ID Kosong", domain.SetProductMaterialsInput{ProductID: "", Items: []domain.ProductMaterialItemInput{{MaterialID: "mat-1", QtyPerUnit: 1}}}},
			{"Material ID Kosong", domain.SetProductMaterialsInput{ProductID: productID, Items: []domain.ProductMaterialItemInput{{MaterialID: "", QtyPerUnit: 1}}}},
			{"Qty Nol", domain.SetProductMaterialsInput{ProductID: productID, Items: []domain.ProductMaterialItemInput{{MaterialID: "mat-1", QtyPerUnit: 0}}}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.SetProductMaterials(context.Background(), tc.input)
				assert.Error(t, err)
				assert.Nil(t, result)
			})
		}
	})
}

func TestProductMaterialUsecase_GetProductMaterials(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockPMRepo, _, _, uc := setupProductMaterialTest()
		productID := "prod-123"
		expected := []domain.ProductMaterial{{ID: "pm-1", ProductID: productID}}

		mockPMRepo.On("FetchByProduct", mock.Anything, productID).Return(expected, nil).Once()

		result, err := uc.GetProductMaterials(context.Background(), productID)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockPMRepo.AssertExpectations(t)
	})

	t.Run("Failed - Product ID Kosong", func(t *testing.T) {
		_, _, _, uc := setupProductMaterialTest()

		result, err := uc.GetProductMaterials(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestProductMaterialUsecase_CalculateMaterialCost(t *testing.T) {
	t.Run("Success - Hitung Total Biaya Material", func(t *testing.T) {
		mockPMRepo, _, _, uc := setupProductMaterialTest()
		productID := "prod-123"
		orderQty := 20

		items := []domain.ProductMaterial{
			{QtyPerUnit: 1.2, Material: &domain.Material{UnitPrice: 25000}}, // 1.2 * 25000 * 20 = 600.000
			{QtyPerUnit: 3, Material: &domain.Material{UnitPrice: 1000}},    // 3 * 1000 * 20   =  60.000
		}
		mockPMRepo.On("FetchByProduct", mock.Anything, productID).Return(items, nil).Once()

		cost, err := uc.CalculateMaterialCost(context.Background(), productID, orderQty)

		assert.NoError(t, err)
		assert.Equal(t, float64(660000), cost)
		mockPMRepo.AssertExpectations(t)
	})

	t.Run("Sukses Tapi Lewati Baris Resep Yang Material-nya Sudah Dihapus", func(t *testing.T) {
		mockPMRepo, _, _, uc := setupProductMaterialTest()
		productID := "prod-123"

		items := []domain.ProductMaterial{
			{QtyPerUnit: 1, Material: &domain.Material{UnitPrice: 10000}}, // dihitung: 10.000 * 5 = 50.000
			{QtyPerUnit: 5, Material: nil},                                // dilewati, jangan sampai panic
		}
		mockPMRepo.On("FetchByProduct", mock.Anything, productID).Return(items, nil).Once()

		cost, err := uc.CalculateMaterialCost(context.Background(), productID, 5)

		assert.NoError(t, err)
		assert.Equal(t, float64(50000), cost)
		mockPMRepo.AssertExpectations(t)
	})

	t.Run("Failed - Repo Error", func(t *testing.T) {
		mockPMRepo, _, _, uc := setupProductMaterialTest()
		productID := "prod-123"

		mockPMRepo.On("FetchByProduct", mock.Anything, productID).Return(nil, errors.New("db down")).Once()

		cost, err := uc.CalculateMaterialCost(context.Background(), productID, 10)

		assert.Error(t, err)
		assert.Equal(t, float64(0), cost)
		mockPMRepo.AssertExpectations(t)
	})

	t.Run("Success - Hitung Biaya Material Dengan Dynamic Fabric", func(t *testing.T) {
		mockPMRepo, mockProductRepo, _, uc := setupProductMaterialTest()
		productID := "prod-123"
		fabricID := "mat-fabric-1"
		orderQty := 10

		items := []domain.ProductMaterial{
			{QtyPerUnit: 4, Material: &domain.Material{UnitPrice: 500}}, // 4 * 500 * 10 = 20.000
		}
		mockPMRepo.On("FetchByProduct", mock.Anything, productID).Return(items, nil).Once()

		mockProduct := &domain.Product{
			ID: productID,
			Fabrics: []domain.ProductFabric{
				{
					FabricID:   &fabricID,
					QtyPerUnit: 1.5,
					Fabric: &domain.Material{
						UnitPrice: 30000, // 1.5 * 30.000 * 10 = 450.000
					},
				},
			},
		}
		mockProductRepo.On("GetByID", mock.Anything, productID).Return(mockProduct, nil).Once()

		cost, err := uc.CalculateMaterialCost(context.Background(), productID, orderQty, fabricID)

		assert.NoError(t, err)
		assert.Equal(t, float64(470000), cost) // 20.000 + 450.000 = 470.000
		mockPMRepo.AssertExpectations(t)
		mockProductRepo.AssertExpectations(t)
	})
}
