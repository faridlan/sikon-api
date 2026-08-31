package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/faridlan/sikon-api/internal/domain"
	domainmocks "github.com/faridlan/sikon-api/internal/domain/mocks"
	"github.com/faridlan/sikon-api/internal/usecase"
)

func newAttendanceUsecaseForTest(attendanceRepo domain.AttendanceRepository, workerRepo domain.WorkerRepository) domain.AttendanceUsecase {
	return usecase.NewAttendanceUsecase(attendanceRepo, workerRepo, 2*time.Second)
}

func TestAttendanceUsecase_RecordAttendance(t *testing.T) {
	attendanceDate := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)

	t.Run("success - calculates daily amount", func(t *testing.T) {
		attendanceRepo := new(domainmocks.AttendanceRepository)
		workerRepo := new(domainmocks.WorkerRepository)
		uc := newAttendanceUsecaseForTest(attendanceRepo, workerRepo)
		workerRepo.On("GetByID", mock.Anything, "worker-1").Return(&domain.Worker{ID: "worker-1", DailyRate: 100000}, nil).Once()
		attendanceRepo.On("GetByWorkerAndDate", mock.Anything, "worker-1", attendanceDate).Return(nil, nil).Once()
		attendanceRepo.On("Create", mock.Anything, mock.MatchedBy(func(att *domain.Attendance) bool {
			return att.WorkDurationIndex == 0.5 && att.DailyRate == 100000 && att.TotalAmount == 50000 && att.Status == domain.AttendanceStatusHalfDay
		})).Return(nil).Once()

		result, err := uc.RecordAttendance(context.Background(), domain.AttendanceCreateInput{
			WorkerID: "worker-1", AttendanceDate: attendanceDate, Status: domain.AttendanceStatusHalfDay,
		})

		assert.NoError(t, err)
		assert.Equal(t, 0.5, result.WorkDurationIndex)
		assert.Equal(t, float64(50000), result.TotalAmount)
		attendanceRepo.AssertExpectations(t)
		workerRepo.AssertExpectations(t)
	})

	t.Run("success - manual duration overrides status", func(t *testing.T) {
		attendanceRepo := new(domainmocks.AttendanceRepository)
		workerRepo := new(domainmocks.WorkerRepository)
		uc := newAttendanceUsecaseForTest(attendanceRepo, workerRepo)
		workerRepo.On("GetByID", mock.Anything, "worker-1").Return(&domain.Worker{ID: "worker-1", DailyRate: 80000}, nil).Once()
		attendanceRepo.On("GetByWorkerAndDate", mock.Anything, "worker-1", attendanceDate).Return(nil, nil).Once()
		attendanceRepo.On("Create", mock.Anything, mock.MatchedBy(func(att *domain.Attendance) bool {
			return att.WorkDurationIndex == 0.25 && att.TotalAmount == 20000
		})).Return(nil).Once()

		result, err := uc.RecordAttendance(context.Background(), domain.AttendanceCreateInput{
			WorkerID: "worker-1", AttendanceDate: attendanceDate, Status: domain.AttendanceStatusPresent, WorkDurationIndex: 0.25,
		})

		assert.NoError(t, err)
		assert.Equal(t, 0.25, result.WorkDurationIndex)
		assert.Equal(t, float64(20000), result.TotalAmount)
	})

	t.Run("failure - invalid input", func(t *testing.T) {
		attendanceRepo := new(domainmocks.AttendanceRepository)
		uc := newAttendanceUsecaseForTest(attendanceRepo, new(domainmocks.WorkerRepository))

		result, err := uc.RecordAttendance(context.Background(), domain.AttendanceCreateInput{})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("failure - duplicate attendance", func(t *testing.T) {
		attendanceRepo := new(domainmocks.AttendanceRepository)
		workerRepo := new(domainmocks.WorkerRepository)
		uc := newAttendanceUsecaseForTest(attendanceRepo, workerRepo)
		workerRepo.On("GetByID", mock.Anything, "worker-1").Return(&domain.Worker{DailyRate: 100000}, nil).Once()
		attendanceRepo.On("GetByWorkerAndDate", mock.Anything, "worker-1", attendanceDate).Return(&domain.Attendance{ID: "existing"}, nil).Once()

		result, err := uc.RecordAttendance(context.Background(), domain.AttendanceCreateInput{WorkerID: "worker-1", AttendanceDate: attendanceDate})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("failure - worker repository error", func(t *testing.T) {
		attendanceRepo := new(domainmocks.AttendanceRepository)
		workerRepo := new(domainmocks.WorkerRepository)
		uc := newAttendanceUsecaseForTest(attendanceRepo, workerRepo)
		repoErr := errors.New("worker unavailable")
		workerRepo.On("GetByID", mock.Anything, "worker-1").Return(nil, repoErr).Once()

		result, err := uc.RecordAttendance(context.Background(), domain.AttendanceCreateInput{WorkerID: "worker-1", AttendanceDate: attendanceDate})

		assert.ErrorIs(t, err, repoErr)
		assert.Nil(t, result)
	})
}

func TestAttendanceUsecase_RecordBatchAttendance(t *testing.T) {
	date := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)

	t.Run("success - creates eligible workers", func(t *testing.T) {
		attendanceRepo := new(domainmocks.AttendanceRepository)
		workerRepo := new(domainmocks.WorkerRepository)
		uc := newAttendanceUsecaseForTest(attendanceRepo, workerRepo)
		workerRepo.On("GetByID", mock.Anything, "worker-1").Return(&domain.Worker{DailyRate: 100000}, nil).Once()
		workerRepo.On("GetByID", mock.Anything, "worker-2").Return(nil, errors.New("missing")).Once()
		attendanceRepo.On("GetByWorkerAndDate", mock.Anything, "worker-1", date).Return(nil, nil).Once()
		attendanceRepo.On("GetByWorkerAndDate", mock.Anything, "worker-2", date).Return(nil, nil).Once()
		attendanceRepo.On("CreateBatch", mock.Anything, mock.MatchedBy(func(items []domain.Attendance) bool {
			return len(items) == 1 && items[0].WorkerID == "worker-1" && items[0].TotalAmount == 100000
		})).Return(nil).Once()

		result, err := uc.RecordBatchAttendance(context.Background(), domain.BatchAttendanceInput{
			AttendanceDate: date,
			Items:          []domain.BatchAttendanceItem{{WorkerID: "worker-1", Status: domain.AttendanceStatusPresent}, {WorkerID: "worker-2", Status: domain.AttendanceStatusPresent}},
		})

		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("failure - empty items", func(t *testing.T) {
		uc := newAttendanceUsecaseForTest(new(domainmocks.AttendanceRepository), new(domainmocks.WorkerRepository))
		result, err := uc.RecordBatchAttendance(context.Background(), domain.BatchAttendanceInput{AttendanceDate: date})
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAttendanceUsecase_GetListUpdateDelete(t *testing.T) {
	t.Run("get and list success", func(t *testing.T) {
		attendanceRepo := new(domainmocks.AttendanceRepository)
		uc := newAttendanceUsecaseForTest(attendanceRepo, new(domainmocks.WorkerRepository))
		attendanceRepo.On("GetByID", mock.Anything, "att-1").Return(&domain.Attendance{ID: "att-1"}, nil).Once()
		attendanceRepo.On("Fetch", mock.Anything, domain.AttendanceFilter{}, 10, 0).Return([]domain.Attendance{{ID: "att-1"}}, int64(1), nil).Once()

		got, err := uc.GetAttendance(context.Background(), "att-1")
		assert.NoError(t, err)
		assert.Equal(t, "att-1", got.ID)
		list, meta, err := uc.ListAttendances(context.Background(), domain.PaginationQuery{Page: 0, Limit: 101}, domain.AttendanceFilter{})
		assert.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, 1, meta.CurrentPage)
	})

	t.Run("update success and delete failure when paid", func(t *testing.T) {
		attendanceRepo := new(domainmocks.AttendanceRepository)
		uc := newAttendanceUsecaseForTest(attendanceRepo, new(domainmocks.WorkerRepository))
		attendanceRepo.On("GetByID", mock.Anything, "att-1").Return(&domain.Attendance{ID: "att-1", Status: domain.AttendanceStatusPresent, DailyRate: 100000}, nil).Once()
		index := 0.5
		attendanceRepo.On("Update", mock.Anything, mock.MatchedBy(func(att *domain.Attendance) bool { return att.WorkDurationIndex == index && att.TotalAmount == 50000 })).Return(nil).Once()
		updated, err := uc.UpdateAttendance(context.Background(), "att-1", domain.AttendanceUpdateInput{WorkDurationIndex: &index})
		assert.NoError(t, err)
		assert.Equal(t, float64(50000), updated.TotalAmount)

		payrollID := "payroll-1"
		attendanceRepo.On("GetByID", mock.Anything, "att-paid").Return(&domain.Attendance{PayrollID: &payrollID}, nil).Twice()
		_, err = uc.UpdateAttendance(context.Background(), "att-paid", domain.AttendanceUpdateInput{})
		assert.Error(t, err)
		assert.Error(t, uc.DeleteAttendance(context.Background(), "att-paid"))
	})

	t.Run("delete success and empty id failures", func(t *testing.T) {
		attendanceRepo := new(domainmocks.AttendanceRepository)
		uc := newAttendanceUsecaseForTest(attendanceRepo, new(domainmocks.WorkerRepository))
		attendanceRepo.On("GetByID", mock.Anything, "att-1").Return(&domain.Attendance{ID: "att-1"}, nil).Once()
		attendanceRepo.On("Delete", mock.Anything, "att-1").Return(nil).Once()
		assert.NoError(t, uc.DeleteAttendance(context.Background(), "att-1"))
		_, err := uc.GetAttendance(context.Background(), "")
		assert.Error(t, err)
		_, err = uc.UpdateAttendance(context.Background(), "", domain.AttendanceUpdateInput{})
		assert.Error(t, err)
		assert.Error(t, uc.DeleteAttendance(context.Background(), ""))
	})
}
