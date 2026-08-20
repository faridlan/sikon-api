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

func TestPayrollUsecase_CreatePayroll(t *testing.T) {
	t.Run("Success - Create Draft Payroll", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		workLogIDs := []string{"wl-uuid-1", "wl-uuid-2"}
		input := domain.PayrollCreateInput{
			StartDate:   time.Now().AddDate(0, 0, -7),
			EndDate:     time.Now(),
			WorkLogIDs:  workLogIDs,
			CreatedByID: "user-uuid-1",
		}

		mockPayrollRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Payroll"), workLogIDs).Return(nil).Once()

		mockCreatedPayroll := &domain.Payroll{
			ID:            "payroll-uuid-1",
			PayrollNumber: "PAY-202608-001",
			TotalAmount:   1500000,
			Status:        domain.PayrollStatusDraft,
		}
		mockPayrollRepo.On("GetByID", mock.Anything, mock.Anything).Return(mockCreatedPayroll, nil).Once()

		result, err := uc.CreatePayroll(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, domain.PayrollStatusDraft, result.Status)
		assert.Equal(t, float64(1500000), result.TotalAmount)

		mockPayrollRepo.AssertExpectations(t)
	})

	t.Run("Error - Empty WorkLog IDs", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		input := domain.PayrollCreateInput{
			StartDate:  time.Now().AddDate(0, 0, -7),
			EndDate:    time.Now(),
			WorkLogIDs: []string{}, // Kosong
		}

		result, err := uc.CreatePayroll(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Error - Invalid Date Range", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		input := domain.PayrollCreateInput{
			StartDate:  time.Time{}, // Zero time
			EndDate:    time.Now(),
			WorkLogIDs: []string{"wl-1"},
		}

		result, err := uc.CreatePayroll(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPayrollUsecase_GetPayroll(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		payrollID := "payroll-uuid-1"
		mockPayroll := &domain.Payroll{
			ID:            payrollID,
			PayrollNumber: "PAY-202608-001",
			TotalAmount:   2000000,
		}

		mockPayrollRepo.On("GetByID", mock.Anything, payrollID).Return(mockPayroll, nil).Once()

		result, err := uc.GetPayroll(context.Background(), payrollID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "PAY-202608-001", result.PayrollNumber)

		mockPayrollRepo.AssertExpectations(t)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		result, err := uc.GetPayroll(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPayrollUsecase_ListPayrolls(t *testing.T) {
	t.Run("Success - Fetch Data With Pagination", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		query := domain.PaginationQuery{Page: 1, Limit: 10}
		filter := domain.PayrollFilter{Status: domain.PayrollStatusDraft}

		mockPayrolls := []domain.Payroll{
			{ID: "p-1", PayrollNumber: "PAY-202608-001", Status: domain.PayrollStatusDraft},
		}

		mockPayrollRepo.On("Fetch", mock.Anything, filter, 10, 0).Return(mockPayrolls, int64(1), nil).Once()

		results, meta, err := uc.ListPayrolls(context.Background(), query, filter)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, int64(1), meta.TotalItems)
		assert.Equal(t, 1, meta.CurrentPage)

		mockPayrollRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failed", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		query := domain.PaginationQuery{Page: 1, Limit: 10}
		filter := domain.PayrollFilter{}
		mockErr := errors.New("database error")

		mockPayrollRepo.On("Fetch", mock.Anything, filter, 10, 0).Return(nil, int64(0), mockErr).Once()

		results, _, err := uc.ListPayrolls(context.Background(), query, filter)

		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Equal(t, mockErr, err)

		mockPayrollRepo.AssertExpectations(t)
	})
}

func TestPayrollUsecase_ProcessPayrollPayment(t *testing.T) {
	t.Run("Success - Process Payment & Auto Create Expense HPP", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		payrollID := "payroll-uuid-1"
		operatorID := "acc-user-uuid"

		poID := "batch-po-123"
		mockPayroll := &domain.Payroll{
			ID:            payrollID,
			PayrollNumber: "PAY-202608-001",
			TotalAmount:   1500000,
			Status:        domain.PayrollStatusDraft,
			StartDate:     time.Now().AddDate(0, 0, -7),
			EndDate:       time.Now(),
			WorkLogs: []domain.WorkLog{
				{ID: "wl-1", BatchPoID: &poID},
			},
		}

		mockCategories := []domain.ExpenseCategory{
			{ID: "cat-hpp-1", Name: "Ongkos Jahit/Borongan", Type: domain.ExpenseTypeHPP},
		}

		mockPayrollRepo.On("GetByID", mock.Anything, payrollID).Return(mockPayroll, nil).Once()
		mockExpenseRepo.On("FetchCategories", mock.Anything).Return(mockCategories, nil).Once()
		mockExpenseRepo.On("Create", mock.Anything, mock.MatchedBy(func(exp *domain.Expense) bool {
			return exp.Amount == 1500000 && exp.ExpenseCategoryID == "cat-hpp-1"
		})).Return(nil).Once()
		mockPayrollRepo.On("UpdateStatus", mock.Anything, payrollID, domain.PayrollStatusPaid, mock.Anything, mock.Anything).Return(nil).Once()

		paidPayroll := *mockPayroll
		paidPayroll.Status = domain.PayrollStatusPaid
		mockPayrollRepo.On("GetByID", mock.Anything, payrollID).Return(&paidPayroll, nil).Once()

		result, err := uc.ProcessPayrollPayment(context.Background(), payrollID, operatorID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, domain.PayrollStatusPaid, result.Status)

		mockPayrollRepo.AssertExpectations(t)
		mockExpenseRepo.AssertExpectations(t)
	})

	t.Run("Error - Payroll Already Paid", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		payrollID := "payroll-uuid-paid"
		mockPayroll := &domain.Payroll{
			ID:     payrollID,
			Status: domain.PayrollStatusPaid,
		}

		mockPayrollRepo.On("GetByID", mock.Anything, payrollID).Return(mockPayroll, nil).Once()

		result, err := uc.ProcessPayrollPayment(context.Background(), payrollID, "acc-user-uuid")

		assert.Error(t, err)
		assert.Nil(t, result)
		mockPayrollRepo.AssertExpectations(t)
	})
}

func TestPayrollUsecase_DeletePayroll(t *testing.T) {
	t.Run("Success - Delete Draft Payroll", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		payrollID := "payroll-uuid-1"
		mockPayroll := &domain.Payroll{
			ID:     payrollID,
			Status: domain.PayrollStatusDraft,
		}

		mockPayrollRepo.On("GetByID", mock.Anything, payrollID).Return(mockPayroll, nil).Once()
		mockPayrollRepo.On("Delete", mock.Anything, payrollID).Return(nil).Once()

		err := uc.DeletePayroll(context.Background(), payrollID)

		assert.NoError(t, err)
		mockPayrollRepo.AssertExpectations(t)
	})

	t.Run("Error - Cannot Delete Paid Payroll", func(t *testing.T) {
		mockPayrollRepo := new(mocks.PayrollRepository)
		mockExpenseRepo := new(mocks.ExpenseRepository)
		uc := usecase.NewPayrollUsecase(mockPayrollRepo, mockExpenseRepo, time.Second*2)

		payrollID := "payroll-uuid-paid"
		mockPayroll := &domain.Payroll{
			ID:     payrollID,
			Status: domain.PayrollStatusPaid, // Sudah lunas
		}

		mockPayrollRepo.On("GetByID", mock.Anything, payrollID).Return(mockPayroll, nil).Once()

		err := uc.DeletePayroll(context.Background(), payrollID)

		assert.Error(t, err)
		mockPayrollRepo.AssertExpectations(t)
	})
}
