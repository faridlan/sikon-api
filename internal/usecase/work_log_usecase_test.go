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

func TestWorkLogUsecase_CreateWorkLog(t *testing.T) {
	t.Run("Success - Auto Calculate Total Amount", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		workerID := "worker-uuid-1"
		mockWorker := &domain.Worker{ID: workerID, Name: "Mang Wahyu"}
		mockWorkerRepo.On("GetByID", mock.Anything, workerID).Return(mockWorker, nil).Once()

		input := domain.WorkLogCreateInput{
			WorkerID:    workerID,
			JobType:     domain.JobTypeJahit,
			Qty:         5,
			RatePerQty:  12000,
			WorkDate:    time.Now(),
			CreatedByID: "user-uuid-1",
		}

		mockWorkLogRepo.On("Create", mock.Anything, mock.MatchedBy(func(log *domain.WorkLog) bool {
			return log.TotalAmount == 60000 // 5 pcs * 12.000 = 60.000
		})).Return(nil).Once()

		result, err := uc.CreateWorkLog(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(60000), result.TotalAmount)
		mockWorkerRepo.AssertExpectations(t)
		mockWorkLogRepo.AssertExpectations(t)
	})

	t.Run("Error - Worker Not Found", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		workerID := "invalid-worker"
		mockWorkerRepo.On("GetByID", mock.Anything, workerID).Return(nil, domain.ErrNotFound).Once()

		input := domain.WorkLogCreateInput{
			WorkerID:   workerID,
			Qty:        5,
			RatePerQty: 12000,
			WorkDate:   time.Now(),
		}

		result, err := uc.CreateWorkLog(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockWorkerRepo.AssertExpectations(t)
	})

	t.Run("Error - Bad Param (Invalid Qty Or Rate)", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		input := domain.WorkLogCreateInput{
			WorkerID:   "worker-uuid-1",
			Qty:        0, // Invalid Qty
			RatePerQty: 12000,
			WorkDate:   time.Now(),
		}

		result, err := uc.CreateWorkLog(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWorkLogUsecase_GetWorkLog(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		mockLog := &domain.WorkLog{ID: "wl-uuid-123", Qty: 10, TotalAmount: 120000}
		mockWorkLogRepo.On("GetByID", mock.Anything, "wl-uuid-123").Return(mockLog, nil).Once()

		result, err := uc.GetWorkLog(context.Background(), "wl-uuid-123")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 10, result.Qty)
		mockWorkLogRepo.AssertExpectations(t)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		result, err := uc.GetWorkLog(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWorkLogUsecase_ListWorkLogs(t *testing.T) {
	t.Run("Success - Fetch Data With Pagination", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		query := domain.PaginationQuery{Page: 1, Limit: 10}
		workerID := "worker-uuid-1"
		filter := domain.WorkLogFilter{WorkerID: &workerID, JobType: domain.JobTypeJahit}

		mockLogs := []domain.WorkLog{
			{ID: "wl-1", WorkerID: workerID, Qty: 5, JobType: domain.JobTypeJahit},
		}

		mockWorkLogRepo.On("Fetch", mock.Anything, filter, 10, 0).Return(mockLogs, int64(1), nil).Once()

		results, meta, err := uc.ListWorkLogs(context.Background(), query, filter)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, int64(1), meta.TotalItems)
		assert.Equal(t, 1, meta.CurrentPage)
		mockWorkLogRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		query := domain.PaginationQuery{Page: 1, Limit: 10}
		filter := domain.WorkLogFilter{}
		mockErr := errors.New("database connection error")

		mockWorkLogRepo.On("Fetch", mock.Anything, filter, 10, 0).Return(nil, int64(0), mockErr).Once()

		results, _, err := uc.ListWorkLogs(context.Background(), query, filter)

		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Equal(t, mockErr, err)
		mockWorkLogRepo.AssertExpectations(t)
	})
}

func TestWorkLogUsecase_UpdateWorkLog(t *testing.T) {
	t.Run("Success - Update WorkLog & Recalculate Total Amount", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		logID := "wl-uuid-1"
		existingLog := &domain.WorkLog{
			ID:          logID,
			WorkerID:    "worker-uuid-1",
			JobType:     domain.JobTypeJahit,
			Qty:         5,
			RatePerQty:  12000,
			TotalAmount: 60000,
			WorkDate:    time.Now(),
			PayrollID:   nil, // Belum di-payroll
		}

		input := domain.WorkLogUpdateInput{
			Qty:        10,
			RatePerQty: 15000,
		}

		mockWorkLogRepo.On("GetByID", mock.Anything, logID).Return(existingLog, nil).Once()
		mockWorkLogRepo.On("Update", mock.Anything, mock.MatchedBy(func(l *domain.WorkLog) bool {
			return l.Qty == 10 && l.RatePerQty == 15000 && l.TotalAmount == 150000 // Recalculate 10 * 15000 = 150000
		})).Return(nil).Once()

		result, err := uc.UpdateWorkLog(context.Background(), logID, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(150000), result.TotalAmount)
		mockWorkLogRepo.AssertExpectations(t)
	})

	t.Run("Error - Cannot Update Already Paid WorkLog", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		logID := "wl-uuid-paid"
		payrollID := "payroll-uuid-1"
		existingLog := &domain.WorkLog{
			ID:        logID,
			PayrollID: &payrollID, // Sudah di-payroll
		}

		input := domain.WorkLogUpdateInput{Qty: 10}

		mockWorkLogRepo.On("GetByID", mock.Anything, logID).Return(existingLog, nil).Once()

		result, err := uc.UpdateWorkLog(context.Background(), logID, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockWorkLogRepo.AssertExpectations(t)
	})
}

func TestWorkLogUsecase_DeleteWorkLog(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		logID := "wl-uuid-1"
		existingLog := &domain.WorkLog{ID: logID, PayrollID: nil}

		mockWorkLogRepo.On("GetByID", mock.Anything, logID).Return(existingLog, nil).Once()
		mockWorkLogRepo.On("Delete", mock.Anything, logID).Return(nil).Once()

		err := uc.DeleteWorkLog(context.Background(), logID)

		assert.NoError(t, err)
		mockWorkLogRepo.AssertExpectations(t)
	})

	t.Run("Error - Cannot Delete Already Paid WorkLog", func(t *testing.T) {
		mockWorkLogRepo := new(mocks.WorkLogRepository)
		mockWorkerRepo := new(mocks.WorkerRepository)
		uc := usecase.NewWorkLogUsecase(mockWorkLogRepo, mockWorkerRepo, time.Second*2)

		logID := "wl-uuid-paid"
		payrollID := "payroll-uuid-1"
		existingLog := &domain.WorkLog{ID: logID, PayrollID: &payrollID}

		mockWorkLogRepo.On("GetByID", mock.Anything, logID).Return(existingLog, nil).Once()

		err := uc.DeleteWorkLog(context.Background(), logID)

		assert.Error(t, err)
		mockWorkLogRepo.AssertExpectations(t)
	})
}
