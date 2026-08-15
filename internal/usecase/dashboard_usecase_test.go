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

func TestDashboardUsecase_GetSummary(t *testing.T) {
	mockSummary := &domain.DashboardSummary{
		TotalRevenue:         100000000,
		TotalPaymentReceived: 40000000,
		TotalReceivable:      60000000,
		TotalActiveOrders:    15,
		TotalCompletedOrders: 50,
		TotalCanceledOrders:  2,
	}

	filter := domain.DashboardFilter{
		StartDate: "2026-06-01",
		EndDate:   "2026-06-30",
	}

	t.Run("Success - Get Dashboard Summary (Owner/Global)", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		mockRepo.On("GetSummary", mock.Anything, filter).Return(mockSummary, nil).Once()

		result, err := uc.GetSummary(context.Background(), filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(100000000), result.TotalRevenue)
		assert.Equal(t, int64(15), result.TotalActiveOrders)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Get Dashboard Summary dengan Scoping SalesID", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		salesFilter := domain.DashboardFilter{
			StartDate: "2026-06-01",
			EndDate:   "2026-06-30",
			SalesID:   "sales-uuid-123", // Data Scoping Sales
		}

		mockRepo.On("GetSummary", mock.Anything, salesFilter).Return(mockSummary, nil).Once()

		result, err := uc.GetSummary(context.Background(), salesFilter)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		mockErr := errors.New("database connection lost")
		mockRepo.On("GetSummary", mock.Anything, filter).Return(nil, mockErr).Once()

		result, err := uc.GetSummary(context.Background(), filter)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, mockErr, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Get Dashboard Summary Tanpa Filter", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		emptyFilter := domain.DashboardFilter{}

		mockRepo.On("GetSummary", mock.Anything, emptyFilter).Return(mockSummary, nil).Once()

		result, err := uc.GetSummary(context.Background(), emptyFilter)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockRepo.AssertExpectations(t)
	})
}

func TestDashboardUsecase_GetSalesReport(t *testing.T) {
	mockReport := []domain.SalesReportItem{
		{
			Date:            "2026-06-24",
			TotalRevenue:    220000,
			TotalOrders:     3,
			CompletedOrders: 1,
			CanceledOrders:  1,
		},
		{
			Date:            "2026-06-25",
			TotalRevenue:    110000,
			TotalOrders:     1,
			CompletedOrders: 1,
			CanceledOrders:  0,
		},
	}

	filter := domain.DashboardFilter{
		StartDate: "2026-06-01",
		EndDate:   "2026-06-30",
	}

	t.Run("Success - Get Sales Report (Global)", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		mockRepo.On("GetSalesReport", mock.Anything, filter).Return(mockReport, nil).Once()

		result, err := uc.GetSalesReport(context.Background(), filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		assert.Equal(t, "2026-06-24", result[0].Date)
		assert.Equal(t, float64(220000), result[0].TotalRevenue)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Get Sales Report dengan Scoping SalesID", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		salesFilter := domain.DashboardFilter{
			StartDate: "2026-06-01",
			EndDate:   "2026-06-30",
			SalesID:   "sales-uuid-123",
		}

		mockRepo.On("GetSalesReport", mock.Anything, salesFilter).Return(mockReport, nil).Once()

		result, err := uc.GetSalesReport(context.Background(), salesFilter)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Empty Report", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		emptyReport := []domain.SalesReportItem{}
		mockRepo.On("GetSalesReport", mock.Anything, filter).Return(emptyReport, nil).Once()

		result, err := uc.GetSalesReport(context.Background(), filter)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 0)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		mockErr := errors.New("timeout querying database")
		mockRepo.On("GetSalesReport", mock.Anything, filter).Return(nil, mockErr).Once()

		result, err := uc.GetSalesReport(context.Background(), filter)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, mockErr, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestDashboardUsecase_GetReceivablesReport(t *testing.T) {
	mockReport := []domain.ReceivableReportItem{
		{
			OrderID:       "order-1",
			OrderNumber:   "ORD-001",
			OrderDate:     "2026-06-24",
			CustomerName:  "PT Maju Jaya",
			SalesName:     "Budi Sales",
			OrderStatus:   "production",
			TotalAmount:   150000,
			TotalPaid:     50000,
			RemainingBill: 100000,
		},
	}

	t.Run("Success - Get Receivables Report (Global / Owner)", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		// 🚨 Penyesuaian: Passing string kosong "" untuk Owner/Finance (Melihat seluruh data)
		mockRepo.On("GetReceivablesReport", mock.Anything, "").Return(mockReport, nil).Once()

		result, err := uc.GetReceivablesReport(context.Background(), "")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, "PT Maju Jaya", result[0].CustomerName)
		assert.Equal(t, float64(100000), result[0].RemainingBill)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Get Receivables Report dengan Scoping SalesID", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		salesID := "sales-uuid-123"

		// 🚨 Penyesuaian: Passing salesID untuk filter khusus Sales
		mockRepo.On("GetReceivablesReport", mock.Anything, salesID).Return(mockReport, nil).Once()

		result, err := uc.GetReceivablesReport(context.Background(), salesID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Empty Report", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		emptyReport := []domain.ReceivableReportItem{}
		mockRepo.On("GetReceivablesReport", mock.Anything, "").Return(emptyReport, nil).Once()

		result, err := uc.GetReceivablesReport(context.Background(), "")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 0)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		mockErr := errors.New("database connection timeout")
		mockRepo.On("GetReceivablesReport", mock.Anything, "").Return(nil, mockErr).Once()

		result, err := uc.GetReceivablesReport(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, mockErr, err)

		mockRepo.AssertExpectations(t)
	})
}
