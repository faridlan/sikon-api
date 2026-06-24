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
	// Setup data mock balikan dari repo
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

	t.Run("Success - Get Dashboard Summary", func(t *testing.T) {
		// 1. Inisiasi Mock Repository
		mockRepo := new(mocks.DashboardRepository)

		// 2. Inisiasi Usecase dengan Mock Repo dan Timeout
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		// 3. Set Ekspektasi: Usecase harus memanggil repo.GetSummary dengan filter yang sama persis
		mockRepo.On("GetSummary", mock.Anything, filter).Return(mockSummary, nil).Once()

		// 4. Eksekusi
		result, err := uc.GetSummary(context.Background(), filter)

		// 5. Asersi
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(100000000), result.TotalRevenue)
		assert.Equal(t, int64(15), result.TotalActiveOrders)

		// 6. Pastikan ekspektasi pemanggilan fungsi mock benar-benar terjadi
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockRepo := new(mocks.DashboardRepository)
		uc := usecase.NewDashboardUsecase(mockRepo, time.Second*2)

		// Set Ekspektasi: Repo mengembalikan error (misal db down)
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
