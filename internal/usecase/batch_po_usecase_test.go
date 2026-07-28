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

func setupBatchPOTest() (*mocks.BatchPORepository, domain.BatchPOUsecase) {
	mockRepo := new(mocks.BatchPORepository)
	uc := usecase.NewBatchPOUsecase(mockRepo, 2*time.Second)
	return mockRepo, uc
}

func TestBatchPOUsecase_CreateBatchPO(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		input := domain.BatchPOCreateInput{
			Name:        "PO Juni 2026",
			TargetMonth: 6,    // <-- TAMBAHAN
			TargetYear:  2026, // <-- TAMBAHAN
			StartDate:   time.Now(),
			EndDate:     time.Now().AddDate(0, 0, 7),
			Quota:       100,
		}

		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(b *domain.BatchPO) bool {
			// <-- PASTIKAN MOCK MENGECEK FIELD BARU INI
			return b.Name == input.Name &&
				b.TargetMonth == 6 &&
				b.TargetYear == 2026 &&
				b.Status == domain.BatchPOStatusDraft
		})).Return(nil).Once()

		result, err := uc.CreateBatchPO(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, domain.BatchPOStatusDraft, result.Status)
		assert.Equal(t, 6, result.TargetMonth)
		assert.Equal(t, 2026, result.TargetYear)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repo Failed", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		input := domain.BatchPOCreateInput{Name: "PO Error", TargetMonth: 6, TargetYear: 2026}

		expectedErr := errors.New("database error")
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(expectedErr).Once()

		result, err := uc.CreateBatchPO(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestBatchPOUsecase_GetBatchPO(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		id := "batch-123"
		expectedData := &domain.BatchPO{ID: id, Name: "PO Aktif"}

		mockRepo.On("GetByID", mock.Anything, id).Return(expectedData, nil).Once()

		result, err := uc.GetBatchPO(context.Background(), id)

		assert.NoError(t, err)
		assert.Equal(t, expectedData, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		id := "batch-404"

		mockRepo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetBatchPO(context.Background(), id)

		assert.Error(t, err)
		assert.Nil(t, result)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrNotFound, appErr.ErrType)
		mockRepo.AssertExpectations(t)
	})
}

func TestBatchPOUsecase_ListBatchPOs(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		query := domain.PaginationQuery{Page: 1, Limit: 10}
		expectedData := []domain.BatchPO{{ID: "1"}, {ID: "2"}}

		mockRepo.On("Fetch", mock.Anything, 10, 0).Return(expectedData, int64(15), nil).Once()

		result, meta, err := uc.ListBatchPOs(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 1, meta.CurrentPage)
		assert.Equal(t, 2, meta.TotalPages) // math.Ceil(15/10) = 2
		assert.Equal(t, int64(15), meta.TotalItems)
		mockRepo.AssertExpectations(t)
	})
}

func TestBatchPOUsecase_ListActiveBatchPOs(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		expectedData := []domain.BatchPO{{ID: "1", Status: domain.BatchPOStatusActive}}

		mockRepo.On("FetchActive", mock.Anything).Return(expectedData, nil).Once()

		result, err := uc.ListActiveBatchPOs(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		mockRepo.AssertExpectations(t)
	})
}

func TestBatchPOUsecase_UpdateBatchPO(t *testing.T) {
	t.Run("Success - Partial Update", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		id := "batch-123"
		existingData := &domain.BatchPO{ID: id, Name: "PO Lama", Quota: 50, TargetMonth: 5, TargetYear: 2026}

		newName := "PO Baru"
		newQuota := 200
		newMonth := 7
		newYear := 2026

		input := domain.BatchPOUpdateInput{
			Name:        newName,
			Quota:       &newQuota,
			TargetMonth: &newMonth, // <-- UJI UPDATE BULAN
			TargetYear:  &newYear,
		}

		mockRepo.On("GetByID", mock.Anything, id).Return(existingData, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(b *domain.BatchPO) bool {
			return b.Name == "PO Baru" && b.Quota == 200 && b.TargetMonth == 7 && b.TargetYear == 2026
		})).Return(nil).Once()

		result, err := uc.UpdateBatchPO(context.Background(), id, input)

		assert.NoError(t, err)
		assert.Equal(t, "PO Baru", result.Name)
		assert.Equal(t, 200, result.Quota)
		assert.Equal(t, 7, result.TargetMonth)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		id := "batch-404"
		newName := "PO Baru"
		input := domain.BatchPOUpdateInput{Name: newName}

		mockRepo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.UpdateBatchPO(context.Background(), id, input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestBatchPOUsecase_UpdateStatus(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		id := "batch-123"
		existingData := &domain.BatchPO{ID: id, Status: domain.BatchPOStatusDraft}

		mockRepo.On("GetByID", mock.Anything, id).Return(existingData, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(b *domain.BatchPO) bool {
			return b.Status == domain.BatchPOStatusActive
		})).Return(nil).Once()

		err := uc.UpdateStatus(context.Background(), id, domain.BatchPOStatusActive)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Invalid Status", func(t *testing.T) {
		_, uc := setupBatchPOTest()

		err := uc.UpdateStatus(context.Background(), "batch-123", "status_ngasal")

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
	})
}

func TestBatchPOUsecase_DeleteBatchPO(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		id := "batch-123"

		mockRepo.On("GetByID", mock.Anything, id).Return(&domain.BatchPO{ID: id}, nil).Once()
		mockRepo.On("Delete", mock.Anything, id).Return(nil).Once()

		err := uc.DeleteBatchPO(context.Background(), id)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo, uc := setupBatchPOTest()
		id := "batch-404"

		mockRepo.On("GetByID", mock.Anything, id).Return(nil, domain.ErrNotFound).Once()

		err := uc.DeleteBatchPO(context.Background(), id)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}
