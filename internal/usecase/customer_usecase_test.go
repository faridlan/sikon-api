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

func TestCustomerUsecase_CreateCustomer(t *testing.T) {
	input := domain.CustomerCreateInput{
		Name:      "PT Maju Jaya",
		Phone:     "08123456789",
		Address:   "Jl. Sudirman",
		CreatedBy: "user-123",
	}

	t.Run("Success - Tanpa SalesID", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *domain.Customer) bool {
			return c.Name == input.Name && c.Phone == input.Phone && c.Address == input.Address
		})).Return(nil).Once()

		result, err := uc.CreateCustomer(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		mockRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("Success - Dengan SalesID Valid", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		inputWithSales := input
		inputWithSales.SalesID = "sales-123"

		mockSalesUser := &domain.User{ID: "sales-123", Role: domain.RoleSales}

		// Ekspektasi: Usecase harus mengecek user repo terlebih dahulu
		userRepo.On("GetByID", mock.Anything, "sales-123").Return(mockSalesUser, nil).Once()
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

		result, err := uc.CreateCustomer(context.Background(), inputWithSales)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "sales-123", result.SalesID)
		mockRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("Error - Role Bukan Sales", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		inputWithSales := input
		inputWithSales.SalesID = "admin-123"

		// Simulasi yang ditugaskan ternyata adalah Admin, bukan Sales
		mockAdminUser := &domain.User{ID: "admin-123", Role: domain.RoleAdmin}

		userRepo.On("GetByID", mock.Anything, "admin-123").Return(mockAdminUser, nil).Once()
		// mockRepo.Create TIDAK BOLEH dipanggil jika validasi gagal

		result, err := uc.CreateCustomer(context.Background(), inputWithSales)

		assert.Error(t, err)
		assert.Nil(t, result)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)

		mockRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})
}

func TestCustomerUsecase_GetCustomer(t *testing.T) {
	mockID := "cust-123"
	mockCustomer := &domain.Customer{ID: mockID, Name: "PT Maju Jaya", SalesID: "sales-A"}

	t.Run("Success - Akses Admin (Bebas Lihat)", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockCustomer, nil).Once()

		// Operator adalah Admin, tidak ada restriction SalesID
		result, err := uc.GetCustomer(context.Background(), mockID, "admin-1", domain.RoleAdmin)

		assert.NoError(t, err)
		assert.Equal(t, mockCustomer.Name, result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Akses Sales (Milik Sendiri)", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockCustomer, nil).Once()

		// Operator adalah Sales dan ID-nya sesuai dengan SalesID Customer (sales-A)
		result, err := uc.GetCustomer(context.Background(), mockID, "sales-A", domain.RoleSales)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Forbidden (Akses Sales Milik Orang Lain)", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockCustomer, nil).Once()

		// Operator adalah Sales B, mencoba melihat data Customer milik Sales A
		result, err := uc.GetCustomer(context.Background(), mockID, "sales-B", domain.RoleSales)

		assert.Error(t, err)
		assert.Nil(t, result)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrForbidden, appErr.ErrType)

		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerUsecase_ListCustomers(t *testing.T) {
	query := domain.PaginationQuery{Page: 1, Limit: 10}
	mockCustomers := []domain.Customer{{Name: "Cust A"}, {Name: "Cust B"}}

	t.Run("Success - Akses Admin (Tanpa Filter)", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		// Perhatikan paramter ke-4 adalah "", artinya tidak ada filter SalesID
		mockRepo.On("Fetch", mock.Anything, 10, 0, "").Return(mockCustomers, int64(2), nil).Once()

		customers, meta, err := uc.ListCustomers(context.Background(), query, "admin-123", domain.RoleAdmin)

		assert.NoError(t, err)
		assert.Len(t, customers, 2)
		assert.Equal(t, 1, meta.TotalPages)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Akses Sales (Dengan Filter ID Sendiri)", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		// Perhatikan paramter ke-4 adalah "sales-123", memicu filter di DB
		mockRepo.On("Fetch", mock.Anything, 10, 0, "sales-123").Return(mockCustomers, int64(2), nil).Once()

		customers, meta, err := uc.ListCustomers(context.Background(), query, "sales-123", domain.RoleSales)

		assert.NoError(t, err)
		assert.Len(t, customers, 2)
		assert.Equal(t, 1, meta.TotalPages)
		mockRepo.AssertExpectations(t)
	})
}

// ... (TestCustomerUsecase_UpdateCustomer dan TestCustomerUsecase_DeleteCustomer tetap sama,
// namun pastikan untuk memindahkan inisialisasi mockRepo & userRepo ke dalam masing-masing t.Run()
// sama seperti fungsi-fungsi di atas agar mencegah bentrok antar test)

func TestCustomerUsecase_UpdateCustomer(t *testing.T) {
	mockID := "cust-123"
	existingCust := &domain.Customer{ID: mockID, Name: "PT Lama", Phone: "111"}
	input := domain.CustomerUpdateInput{Name: "PT Baru"}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingCust, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *domain.Customer) bool {
			return c.Name == "PT Baru" && c.Phone == "111"
		})).Return(nil).Once()

		result, err := uc.UpdateCustomer(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "PT Baru", result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		mockRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.UpdateCustomer(context.Background(), mockID, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerUsecase_DeleteCustomer(t *testing.T) {
	mockID := "cust-123"
	existingCust := &domain.Customer{ID: mockID, Name: "PT Lama", Phone: "111"}

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingCust, nil).Once()
		mockRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()
		err := uc.DeleteCustomer(context.Background(), mockID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
