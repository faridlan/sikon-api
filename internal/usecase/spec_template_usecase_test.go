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

func TestSpecTemplateUsecase_CreateSpecTemplate(t *testing.T) {
	mockRepo := new(mocks.SpecTemplateRepository)
	uc := usecase.NewSpecTemplateUsecase(mockRepo, time.Second*2)

	input := domain.SpecTemplateCreateInput{
		Name: "Bahan Rompi Standar",
		Spec: "Drill Halus, Furing Peles",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(s *domain.SpecTemplate) bool {
			return s.Name == input.Name && s.Spec == input.Spec
		})).Return(nil).Once()

		result, err := uc.CreateSpecTemplate(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		assert.Equal(t, input.Spec, result.Spec)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		result, err := uc.CreateSpecTemplate(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestSpecTemplateUsecase_GetSpecTemplate(t *testing.T) {
	mockRepo := new(mocks.SpecTemplateRepository)
	uc := usecase.NewSpecTemplateUsecase(mockRepo, time.Second*2)

	mockID := "spec-123"
	mockSpec := &domain.SpecTemplate{ID: mockID, Name: "Bahan Rompi", Spec: "Drill"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockSpec, nil).Once()

		result, err := uc.GetSpecTemplate(context.Background(), mockID)

		assert.NoError(t, err)
		assert.Equal(t, mockSpec.Name, result.Name)
		assert.Equal(t, mockSpec.Spec, result.Spec)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		_, err := uc.GetSpecTemplate(context.Background(), mockID)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrNotFound, appErr.ErrType)
		mockRepo.AssertExpectations(t)
	})
}

func TestSpecTemplateUsecase_UpdateSpecTemplate(t *testing.T) {
	mockRepo := new(mocks.SpecTemplateRepository)
	uc := usecase.NewSpecTemplateUsecase(mockRepo, time.Second*2)

	mockID := "spec-123"
	existingSpec := &domain.SpecTemplate{ID: mockID, Name: "Bahan Rompi", Spec: "Drill"}
	input := domain.SpecTemplateUpdateInput{Name: "Bahan Rompi Premium", Spec: "Drill Premium"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingSpec, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(s *domain.SpecTemplate) bool {
			return s.Name == "Bahan Rompi Premium" && s.Spec == "Drill Premium"
		})).Return(nil).Once()

		result, err := uc.UpdateSpecTemplate(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "Bahan Rompi Premium", result.Name)
		assert.Equal(t, "Drill Premium", result.Spec)
		mockRepo.AssertExpectations(t)
	})

	// Anda bisa menambahkan test case "Error - Not Found" di sini jika mau
}

func TestSpecTemplateUsecase_ListSpecTemplates(t *testing.T) {
	mockRepo := new(mocks.SpecTemplateRepository)
	uc := usecase.NewSpecTemplateUsecase(mockRepo, time.Second*2)

	query := domain.PaginationQuery{Page: 1, Limit: 10}
	mockSpecs := []domain.SpecTemplate{{Name: "A", Spec: "Spec A"}, {Name: "B", Spec: "Spec B"}}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Fetch", mock.Anything, 10, 0).Return(mockSpecs, int64(2), nil).Once()

		specs, meta, err := uc.ListSpecTemplates(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, specs, 2)
		assert.Equal(t, 1, meta.TotalPages)
		mockRepo.AssertExpectations(t)
	})
}

func TestSpecTemplateUsecase_DeleteSpecTemplate(t *testing.T) {
	mockRepo := new(mocks.SpecTemplateRepository)
	uc := usecase.NewSpecTemplateUsecase(mockRepo, time.Second*2)
	mockID := "spec-123"
	existingSpec := &domain.SpecTemplate{ID: mockID, Name: "Bahan Rompi", Spec: "Drill"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingSpec, nil).Once()
		mockRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()
		err := uc.DeleteSpecTemplate(context.Background(), mockID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
