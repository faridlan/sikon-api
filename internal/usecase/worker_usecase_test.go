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

func TestWorkerUsecase_CreateWorker(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		input := domain.WorkerCreateInput{
			Name:       "Mang Ade",
			Phone:      "081234567890",
			Role:       domain.WorkerRoleTailor,
			SalaryType: domain.WorkerSalaryTypePieceRate,
		}

		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Worker")).Return(nil).Once()

		result, err := uc.CreateWorker(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Mang Ade", result.Name)
		assert.Equal(t, domain.WorkerStatusActive, result.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Bad Param (Empty Name)", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		input := domain.WorkerCreateInput{
			Name:       "",
			Role:       domain.WorkerRoleTailor,
			SalaryType: domain.WorkerSalaryTypePieceRate,
		}

		result, err := uc.CreateWorker(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Error - Bad Param (Invalid Role)", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		input := domain.WorkerCreateInput{
			Name:       "Mang Ade",
			Role:       "invalid_role",
			SalaryType: domain.WorkerSalaryTypePieceRate,
		}

		result, err := uc.CreateWorker(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Error - Bad Param (Invalid Salary Type)", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		input := domain.WorkerCreateInput{
			Name:       "Mang Ade",
			Role:       domain.WorkerRoleTailor,
			SalaryType: "invalid_salary_type",
		}

		result, err := uc.CreateWorker(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWorkerUsecase_GetWorker(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		mockWorker := &domain.Worker{ID: "worker-uuid-123", Name: "Mang Ade"}
		mockRepo.On("GetByID", mock.Anything, "worker-uuid-123").Return(mockWorker, nil).Once()

		result, err := uc.GetWorker(context.Background(), "worker-uuid-123")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Mang Ade", result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		result, err := uc.GetWorker(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWorkerUsecase_ListWorkers(t *testing.T) {
	t.Run("Success - Fetch Data With Pagination", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		query := domain.PaginationQuery{Page: 1, Limit: 10}
		filter := domain.WorkerFilter{Search: "Ade", Role: domain.WorkerRoleTailor}

		mockWorkers := []domain.Worker{
			{ID: "w-1", Name: "Mang Ade", Role: domain.WorkerRoleTailor, Status: domain.WorkerStatusActive},
		}

		mockRepo.On("Fetch", mock.Anything, filter, 10, 0).Return(mockWorkers, int64(1), nil).Once()

		results, meta, err := uc.ListWorkers(context.Background(), query, filter)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, int64(1), meta.TotalItems)
		assert.Equal(t, 1, meta.CurrentPage)
		assert.Equal(t, 1, meta.TotalPages)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Fallback Default Pagination", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		invalidQuery := domain.PaginationQuery{Page: 0, Limit: 0}
		filter := domain.WorkerFilter{}

		mockRepo.On("Fetch", mock.Anything, filter, 10, 0).Return([]domain.Worker{}, int64(0), nil).Once()

		results, meta, err := uc.ListWorkers(context.Background(), invalidQuery, filter)

		assert.NoError(t, err)
		assert.Empty(t, results)
		assert.Equal(t, 1, meta.CurrentPage)
		assert.Equal(t, 10, meta.Limit)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		query := domain.PaginationQuery{Page: 1, Limit: 10}
		filter := domain.WorkerFilter{}
		mockErr := errors.New("database query error")

		mockRepo.On("Fetch", mock.Anything, filter, 10, 0).Return(nil, int64(0), mockErr).Once()

		results, _, err := uc.ListWorkers(context.Background(), query, filter)

		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Equal(t, mockErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestWorkerUsecase_UpdateWorker(t *testing.T) {
	t.Run("Success - Full Update", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		workerID := "worker-uuid-1"
		existingWorker := &domain.Worker{
			ID:         workerID,
			Name:       "Mang Ade",
			Phone:      "081234567890",
			Role:       domain.WorkerRoleTailor,
			SalaryType: domain.WorkerSalaryTypePieceRate,
			Status:     domain.WorkerStatusActive,
		}

		input := domain.WorkerUpdateInput{
			Name:       "Mang Ade Supriatna",
			Phone:      "081299998888",
			Role:       domain.WorkerRoleTailor,
			SalaryType: domain.WorkerSalaryTypePieceRate,
			Status:     domain.WorkerStatusInactive,
		}

		mockRepo.On("GetByID", mock.Anything, workerID).Return(existingWorker, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(w *domain.Worker) bool {
			return w.Name == "Mang Ade Supriatna" && w.Status == domain.WorkerStatusInactive
		})).Return(nil).Once()

		result, err := uc.UpdateWorker(context.Background(), workerID, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Mang Ade Supriatna", result.Name)
		assert.Equal(t, domain.WorkerStatusInactive, result.Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		result, err := uc.UpdateWorker(context.Background(), "", domain.WorkerUpdateInput{})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Error - Worker Not Found", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		workerID := "invalid-uuid"
		mockRepo.On("GetByID", mock.Anything, workerID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.UpdateWorker(context.Background(), workerID, domain.WorkerUpdateInput{Name: "Mang Ade"})

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestWorkerUsecase_DeleteWorker(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		workerID := "worker-uuid-1"
		mockRepo.On("Delete", mock.Anything, workerID).Return(nil).Once()

		err := uc.DeleteWorker(context.Background(), workerID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		err := uc.DeleteWorker(context.Background(), "")

		assert.Error(t, err)
	})

	t.Run("Error - Worker Not Found On Delete", func(t *testing.T) {
		mockRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkerUsecase(mockRepo, time.Second*2)

		workerID := "invalid-uuid"
		mockRepo.On("Delete", mock.Anything, workerID).Return(domain.ErrNotFound).Once()

		err := uc.DeleteWorker(context.Background(), workerID)

		assert.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
		mockRepo.AssertExpectations(t)
	})
}
