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

func setupExpenseTest() (*mocks.ExpenseRepository, domain.ExpenseUsecase) {
	mockRepo := new(mocks.ExpenseRepository)
	uc := usecase.NewExpenseUsecase(mockRepo, 2*time.Second)
	return mockRepo, uc
}

// ==========================================
// TESTS: EXPENSE CATEGORY
// ==========================================

func TestExpenseUsecase_CreateCategory(t *testing.T) {
	t.Run("Success - HPP", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		input := domain.ExpenseCategoryCreateInput{
			Name:        "Bahan Baku",
			Type:        "HPP",
			Description: "Kain dll",
		}

		mockRepo.On("CreateCategory", mock.Anything, mock.MatchedBy(func(cat *domain.ExpenseCategory) bool {
			return cat.Name == "Bahan Baku" && cat.Type == domain.ExpenseTypeHPP
		})).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*domain.ExpenseCategory)
			arg.ID = "cat-123" // Simulate DB assigning ID
		}).Once()

		result, err := uc.CreateCategory(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "cat-123", result.ID)
		assert.Equal(t, domain.ExpenseTypeHPP, result.Type)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed - Invalid Type", func(t *testing.T) {
		_, uc := setupExpenseTest()
		input := domain.ExpenseCategoryCreateInput{
			Name: "Bahan Baku",
			Type: "NGASAL", // Invalid
		}

		result, err := uc.CreateCategory(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
	})

	t.Run("Failed - Empty Name", func(t *testing.T) {
		_, uc := setupExpenseTest()
		input := domain.ExpenseCategoryCreateInput{Name: "", Type: "HPP"}

		result, err := uc.CreateCategory(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Failed - Repo Error", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		input := domain.ExpenseCategoryCreateInput{Name: "Bahan Baku", Type: "HPP"}

		expectedErr := errors.New("db error")
		mockRepo.On("CreateCategory", mock.Anything, mock.Anything).Return(expectedErr).Once()

		result, err := uc.CreateCategory(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestExpenseUsecase_ListCategories(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		expectedData := []domain.ExpenseCategory{
			{ID: "cat-1", Name: "Kain", Type: domain.ExpenseTypeHPP},
		}

		mockRepo.On("FetchCategories", mock.Anything).Return(expectedData, nil).Once()

		result, err := uc.ListCategories(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Kain", result[0].Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed - Repo Error", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		mockRepo.On("FetchCategories", mock.Anything).Return(nil, errors.New("db down")).Once()

		result, err := uc.ListCategories(context.Background())

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

// ==========================================
// TESTS: EXPENSE TRANSACTION
// ==========================================

func TestExpenseUsecase_CreateExpense(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		input := domain.ExpenseCreateInput{
			ExpenseCategoryID: "cat-123",
			Title:             "Beli Kain Ripstop",
			Amount:            500000,
			ExpenseDate:       time.Now(),
		}

		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(exp *domain.Expense) bool {
			return exp.Title == "Beli Kain Ripstop" && exp.Amount == 500000
		})).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*domain.Expense)
			arg.ID = "exp-123"
		}).Once()

		result, err := uc.CreateExpense(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "exp-123", result.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed - Validation Errors", func(t *testing.T) {
		_, uc := setupExpenseTest()

		// Test cases table
		tests := []struct {
			name  string
			input domain.ExpenseCreateInput
		}{
			{"Empty Title", domain.ExpenseCreateInput{Title: "", Amount: 1000, ExpenseCategoryID: "cat-1", ExpenseDate: time.Now()}},
			{"Zero Amount", domain.ExpenseCreateInput{Title: "Title", Amount: 0, ExpenseCategoryID: "cat-1", ExpenseDate: time.Now()}},
			{"Negative Amount", domain.ExpenseCreateInput{Title: "Title", Amount: -100, ExpenseCategoryID: "cat-1", ExpenseDate: time.Now()}},
			{"Empty Category", domain.ExpenseCreateInput{Title: "Title", Amount: 1000, ExpenseCategoryID: "", ExpenseDate: time.Now()}},
			{"Empty Date", domain.ExpenseCreateInput{Title: "Title", Amount: 1000, ExpenseCategoryID: "cat-1"}}, // default IsZero()
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.CreateExpense(context.Background(), tc.input)
				assert.Error(t, err)
				assert.Nil(t, result)
			})
		}
	})

	t.Run("Failed - Repo Error", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		input := domain.ExpenseCreateInput{
			ExpenseCategoryID: "cat-123",
			Title:             "Beli Kain",
			Amount:            1000,
			ExpenseDate:       time.Now(),
		}

		mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db err")).Once()

		result, err := uc.CreateExpense(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestExpenseUsecase_GetExpense(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		id := "exp-123"
		expectedData := &domain.Expense{ID: id, Title: "Beli Alat"}

		mockRepo.On("GetByID", mock.Anything, id).Return(expectedData, nil).Once()

		result, err := uc.GetExpense(context.Background(), id)

		assert.NoError(t, err)
		assert.Equal(t, expectedData, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed - Empty ID", func(t *testing.T) {
		_, uc := setupExpenseTest()

		result, err := uc.GetExpense(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestExpenseUsecase_ListExpenses(t *testing.T) {
	t.Run("Success - Without Filter", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		query := domain.PaginationQuery{Page: 1, Limit: 10}
		expectedData := []domain.Expense{{ID: "1"}, {ID: "2"}}

		// Offset (1-1)*10 = 0
		mockRepo.On("Fetch", mock.Anything, 10, 0, (*string)(nil), (*time.Time)(nil), (*time.Time)(nil)).
			Return(expectedData, int64(15), nil).Once()

		result, meta, err := uc.ListExpenses(context.Background(), query, nil, nil, nil)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(15), meta.TotalItems)
		assert.Equal(t, 2, meta.TotalPages) // 15/10 ceil = 2
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - With Date Filters", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		query := domain.PaginationQuery{Page: 2, Limit: 5} // Offset = 5

		startDate := "2026-07-01"
		endDate := "2026-07-31"

		mockRepo.On("Fetch", mock.Anything, 5, 5, (*string)(nil), mock.AnythingOfType("*time.Time"), mock.AnythingOfType("*time.Time")).
			Return([]domain.Expense{}, int64(0), nil).Once()

		_, _, err := uc.ListExpenses(context.Background(), query, nil, &startDate, &endDate)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestExpenseUsecase_DeleteExpense(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo, uc := setupExpenseTest()
		id := "exp-123"

		mockRepo.On("Delete", mock.Anything, id).Return(nil).Once()

		err := uc.DeleteExpense(context.Background(), id)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failed - Empty ID", func(t *testing.T) {
		_, uc := setupExpenseTest()

		err := uc.DeleteExpense(context.Background(), "")

		assert.Error(t, err)
	})
}
