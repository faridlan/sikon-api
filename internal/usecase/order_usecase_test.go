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

func setupOrderTest() (*mocks.OrderRepository, *mocks.CustomerRepository, *mocks.UserRepository, *mocks.ProductRepository, domain.OrderUsecase) {
	mockOrderRepo := new(mocks.OrderRepository)
	mockCustomerRepo := new(mocks.CustomerRepository)
	mockUserRepo := new(mocks.UserRepository)
	mockProductRepo := new(mocks.ProductRepository)

	uc := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, time.Second*2)

	return mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, uc
}

func TestOrderUsecase_CreateOrder(t *testing.T) {
	mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, uc := setupOrderTest()

	input := domain.OrderCreateInput{
		CustomerID:   "cust-123",
		SalesID:      "user-123",
		ShippingCost: 50000,
		Items: []domain.OrderItemInput{
			{ProductID: "prod-1", Qty: 2, Price: 0},     // Price 0 -> harus fallback ke BasePrice produk
			{ProductID: "prod-2", Qty: 1, Price: 15000}, // Price > 0 -> pakai price ini
		},
	}

	t.Run("Success", func(t *testing.T) {
		// 1. Mock Customer & User exist
		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(&domain.Customer{ID: input.CustomerID}, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, input.SalesID).Return(&domain.User{ID: input.SalesID}, nil).Once()

		// 2. Mock Product 1 (BasePrice: 50000)
		mockProductRepo.On("GetByID", mock.Anything, "prod-1").Return(&domain.Product{ID: "prod-1", BasePrice: 50000}, nil).Once()
		// 3. Mock Product 2 (BasePrice: 10000 tapi user input 15000)
		mockProductRepo.On("GetByID", mock.Anything, "prod-2").Return(&domain.Product{ID: "prod-2", BasePrice: 10000}, nil).Once()

		// 4. Kalkulasi ekspektasi:
		// Item 1: 2 qty * 50000 (BasePrice) = 100000
		// Item 2: 1 qty * 15000 (Input Price) = 15000
		// TotalAmount = 115000
		mockOrderRepo.On("Create", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.CustomerID == input.CustomerID &&
				o.TotalAmount == 115000 &&
				len(o.Items) == 2 &&
				o.OrderStatus == domain.OrderStatusPending &&
				o.PaymentStatus == domain.PaymentStatusUnpaid
		})).Return(nil).Once()

		err := uc.CreateOrder(context.Background(), input)

		assert.NoError(t, err)
		mockCustomerRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Error - Customer Not Found", func(t *testing.T) {
		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(nil, domain.ErrNotFound).Once()

		err := uc.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, "Customer tidak ditemukan", appErr.Message)
	})
}

func TestOrderUsecase_GetOrder(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	mockID := "ord-123"

	t.Run("Success", func(t *testing.T) {
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(&domain.Order{ID: mockID}, nil).Once()

		result, err := uc.GetOrder(context.Background(), mockID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, mockID, result.ID)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetOrder(context.Background(), mockID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestOrderUsecase_ListOrders(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	query := domain.PaginationQuery{Page: 1, Limit: 10}

	t.Run("Success", func(t *testing.T) {
		mockOrders := []domain.Order{{ID: "1"}, {ID: "2"}}
		mockOrderRepo.On("Fetch", mock.Anything, 10, 0).Return(mockOrders, int64(2), nil).Once()

		orders, meta, err := uc.ListOrders(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, orders, 2)
		assert.Equal(t, 1, meta.TotalPages)
	})
}

func TestOrderUsecase_UpdateOrder(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	mockID := "ord-123"
	input := domain.OrderUpdateInput{
		ShippingCost: 15000,
		CourierName:  "JNE",
	}

	t.Run("Success", func(t *testing.T) {
		existingOrder := &domain.Order{ID: mockID, ShippingCost: 0}
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(existingOrder, nil).Once()

		mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.ShippingCost == 15000 && o.CourierName == "JNE"
		})).Return(nil).Once()

		err := uc.UpdateOrder(context.Background(), mockID, input)

		assert.NoError(t, err)
	})
}

func TestOrderUsecase_DeleteOrder(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	mockID := "ord-123"
	existingOrder := &domain.Order{ID: mockID, ShippingCost: 0}

	t.Run("Success", func(t *testing.T) {
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(existingOrder, nil).Once()
		mockOrderRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()
		err := uc.DeleteOrder(context.Background(), mockID)
		assert.NoError(t, err)
	})
}

func TestOrderUsecase_UpdateOrderStatus(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	mockID := "ord-123"

	t.Run("Success - Valid Status", func(t *testing.T) {
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockID, domain.OrderStatusProduction, domain.PaymentStatus("")).Return(nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusProduction)

		assert.NoError(t, err)
	})

	t.Run("Error - Invalid Status", func(t *testing.T) {
		err := uc.UpdateOrderStatus(context.Background(), mockID, "STATUS_NGAWUR")

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)

		// Pastikan repo tidak dipanggil jika status invalid
		mockOrderRepo.AssertNotCalled(t, "UpdateStatus")
	})
}
