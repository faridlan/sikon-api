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

func TestCategoryUsecase_CreateCategory(t *testing.T) {
	mockRepo := new(mocks.CategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo, time.Second*2)

	input := domain.CategoryCreateInput{
		Name: "Kaos Polos",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *domain.Category) bool {
			return c.Name == input.Name
		})).Return(nil).Once()

		result, err := uc.CreateCategory(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		result, err := uc.CreateCategory(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestCategoryUsecase_GetCategory(t *testing.T) {
	mockRepo := new(mocks.CategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo, time.Second*2)

	mockID := "cat-123"
	mockCat := &domain.Category{ID: mockID, Name: "Kaos"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockCat, nil).Once()

		result, err := uc.GetCategory(context.Background(), mockID)

		assert.NoError(t, err)
		assert.Equal(t, mockCat.Name, result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		_, err := uc.GetCategory(context.Background(), mockID)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrNotFound, appErr.ErrType)
		mockRepo.AssertExpectations(t)
	})
}

func TestCategoryUsecase_UpdateCategory(t *testing.T) {
	mockRepo := new(mocks.CategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo, time.Second*2)

	mockID := "cat-123"
	existingCat := &domain.Category{ID: mockID, Name: "Kaos"}
	input := domain.CategoryUpdateInput{Name: "Kaos Lengan Panjang"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingCat, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *domain.Category) bool {
			return c.Name == "Kaos Lengan Panjang"
		})).Return(nil).Once()

		result, err := uc.UpdateCategory(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "Kaos Lengan Panjang", result.Name)
		mockRepo.AssertExpectations(t)
	})
}

func TestCategoryUsecase_ListCategories(t *testing.T) {
	mockRepo := new(mocks.CategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo, time.Second*2)

	query := domain.PaginationQuery{Page: 1, Limit: 10}
	mockCats := []domain.Category{{Name: "A"}, {Name: "B"}}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Fetch", mock.Anything, 10, 0).Return(mockCats, int64(2), nil).Once()

		cats, meta, err := uc.ListCategories(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, cats, 2)
		assert.Equal(t, 1, meta.TotalPages)
		mockRepo.AssertExpectations(t)
	})
}

func TestCategoryUsecase_DeleteCategory(t *testing.T) {
	mockRepo := new(mocks.CategoryRepository)
	uc := usecase.NewCategoryUsecase(mockRepo, time.Second*2)
	mockID := "cat-123"

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()
		err := uc.DeleteCategory(context.Background(), mockID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
