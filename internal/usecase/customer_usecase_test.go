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
	mockRepo := new(mocks.CustomerRepository)
	uc := usecase.NewCustomerUsecase(mockRepo, time.Second*2)

	input := domain.CustomerCreateInput{
		Name:      "PT Maju Jaya",
		Phone:     "08123456789",
		Address:   "Jl. Sudirman",
		CreatedBy: "user-123",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *domain.Customer) bool {
			return c.Name == input.Name && c.Phone == input.Phone && c.Address == input.Address
		})).Return(nil).Once()

		result, err := uc.CreateCustomer(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		result, err := uc.CreateCustomer(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerUsecase_GetCustomer(t *testing.T) {
	mockRepo := new(mocks.CustomerRepository)
	uc := usecase.NewCustomerUsecase(mockRepo, time.Second*2)

	mockID := "cust-123"
	mockCustomer := &domain.Customer{ID: mockID, Name: "PT Maju Jaya"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockCustomer, nil).Once()

		result, err := uc.GetCustomer(context.Background(), mockID)

		assert.NoError(t, err)
		assert.Equal(t, mockCustomer.Name, result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetCustomer(context.Background(), mockID)

		assert.Error(t, err)
		assert.Nil(t, result)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrNotFound, appErr.ErrType)
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerUsecase_ListCustomers(t *testing.T) {
	mockRepo := new(mocks.CustomerRepository)
	uc := usecase.NewCustomerUsecase(mockRepo, time.Second*2)

	query := domain.PaginationQuery{Page: 1, Limit: 10}
	mockCustomers := []domain.Customer{{Name: "Cust A"}, {Name: "Cust B"}}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Fetch", mock.Anything, 10, 0).Return(mockCustomers, int64(2), nil).Once()

		customers, meta, err := uc.ListCustomers(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, customers, 2)
		assert.Equal(t, 1, meta.TotalPages)
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerUsecase_UpdateCustomer(t *testing.T) {
	mockRepo := new(mocks.CustomerRepository)
	uc := usecase.NewCustomerUsecase(mockRepo, time.Second*2)

	mockID := "cust-123"
	existingCust := &domain.Customer{ID: mockID, Name: "PT Lama", Phone: "111"}
	input := domain.CustomerUpdateInput{Name: "PT Baru"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingCust, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *domain.Customer) bool {
			return c.Name == "PT Baru" && c.Phone == "111" // phone harus tetap sama karena tidak diupdate
		})).Return(nil).Once()

		result, err := uc.UpdateCustomer(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "PT Baru", result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.UpdateCustomer(context.Background(), mockID, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerUsecase_DeleteCustomer(t *testing.T) {
	mockRepo := new(mocks.CustomerRepository)
	uc := usecase.NewCustomerUsecase(mockRepo, time.Second*2)

	mockID := "cust-123"

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()
		err := uc.DeleteCustomer(context.Background(), mockID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
