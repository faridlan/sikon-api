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
		inputWithSales.SalesID = "owner-123"

		// Simulasi yang ditugaskan ternyata adalah Owner/Accounting, bukan Sales
		mockOwnerUser := &domain.User{ID: "owner-123", Role: domain.RoleOwner}

		userRepo.On("GetByID", mock.Anything, "owner-123").Return(mockOwnerUser, nil).Once()

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

	t.Run("Success - Akses Owner (Bebas Lihat)", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		userRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, userRepo, time.Second*2)

		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockCustomer, nil).Once()

		// Operator adalah Owner, tidak ada restriction SalesID
		result, err := uc.GetCustomer(context.Background(), mockID, "owner-1", domain.RoleOwner)

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

	t.Run("Success - Akses Owner (Tanpa Filter)", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		mockUserRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, mockUserRepo, time.Second*2)

		emptyFilter := domain.CustomerFilter{}
		mockRepo.On("Fetch", mock.Anything, emptyFilter, 10, 0).Return(mockCustomers, int64(2), nil).Once()

		customers, meta, err := uc.ListCustomers(context.Background(), query, emptyFilter, "owner-123", domain.RoleOwner)

		assert.NoError(t, err)
		assert.Len(t, customers, 2)
		assert.Equal(t, 1, meta.TotalPages)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Akses Owner (Filter Search & SalesID dari Frontend)", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		mockUserRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, mockUserRepo, time.Second*2)

		inputFilter := domain.CustomerFilter{Search: "Budi", SalesID: "sales-123"}

		// Karena Owner, filter diteruskan utuh ke Repo
		mockRepo.On("Fetch", mock.Anything, inputFilter, 10, 0).Return(mockCustomers, int64(2), nil).Once()

		customers, meta, err := uc.ListCustomers(context.Background(), query, inputFilter, "owner-123", domain.RoleOwner)

		assert.NoError(t, err)
		assert.Len(t, customers, 2)
		mockRepo.AssertExpectations(t)
		assert.Equal(t, 1, meta.TotalPages)
	})

	t.Run("Success - KEAMANAN Sales: Menggagalkan By-Pass SalesID", func(t *testing.T) {
		mockRepo := new(mocks.CustomerRepository)
		mockUserRepo := new(mocks.UserRepository)
		uc := usecase.NewCustomerUsecase(mockRepo, mockUserRepo, time.Second*2)

		// SKENARIO: Sales (ID: "sales-asli-123") bertindak nakal dengan mengubah URL Frontend
		// menjadi ?sales_id=sales-orang-lain-999
		hackerFilter := domain.CustomerFilter{SalesID: "sales-orang-lain-999"}

		// EKSPEKTASI REPO: Usecase HARUS menimpa input hacker dan menggantinya dengan ID Asli!
		expectedFilter := domain.CustomerFilter{SalesID: "sales-asli-123"}

		mockRepo.On("Fetch", mock.Anything, expectedFilter, 10, 0).Return(mockCustomers, int64(2), nil).Once()

		customers, meta, err := uc.ListCustomers(context.Background(), query, hackerFilter, "sales-asli-123", domain.RoleSales)

		assert.NoError(t, err)
		assert.Len(t, customers, 2)
		mockRepo.AssertExpectations(t)
		assert.Equal(t, 1, meta.TotalPages)
	})
}

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
