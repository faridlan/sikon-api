package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/domain/mocks"
	"github.com/faridlan/sikon-api/internal/usecase"
)

func TestBankAccountUsecase_CreateAccount(t *testing.T) {
	mockRepo := new(mocks.BankAccountRepository)
	uc := usecase.NewBankAccountUsecase(mockRepo, time.Second*2)

	userID := "user-123"
	input := domain.BankAccountCreateInput{
		UserID:        &userID,
		BankName:      "BCA",
		AccountNumber: "1234567890",
		AccountName:   "Budi",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(b *domain.BankAccount) bool {
			return b.BankName == input.BankName && b.AccountNumber == input.AccountNumber
		})).Return(nil).Once()

		result, err := uc.CreateAccount(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.BankName, result.BankName)
		mockRepo.AssertExpectations(t)
	})
}

func TestBankAccountUsecase_GetAccount(t *testing.T) {
	mockRepo := new(mocks.BankAccountRepository)
	uc := usecase.NewBankAccountUsecase(mockRepo, time.Second*2)

	mockID := "bank-123"
	mockAccount := &domain.BankAccount{ID: mockID, BankName: "BCA"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockAccount, nil).Once()

		result, err := uc.GetAccount(context.Background(), mockID)

		assert.NoError(t, err)
		assert.Equal(t, mockAccount.BankName, result.BankName)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetAccount(context.Background(), mockID)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestBankAccountUsecase_ListAccounts(t *testing.T) {
	mockRepo := new(mocks.BankAccountRepository)
	uc := usecase.NewBankAccountUsecase(mockRepo, time.Second*2)

	query := domain.PaginationQuery{Page: 1, Limit: 10}
	mockAccounts := []domain.BankAccount{{BankName: "BCA"}, {BankName: "Mandiri"}}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Fetch", mock.Anything, 10, 0).Return(mockAccounts, int64(2), nil).Once()

		accounts, meta, err := uc.ListAccounts(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, accounts, 2)
		assert.Equal(t, 1, meta.TotalPages)
		mockRepo.AssertExpectations(t)
	})
}

func TestBankAccountUsecase_UpdateAccount(t *testing.T) {
	mockRepo := new(mocks.BankAccountRepository)
	uc := usecase.NewBankAccountUsecase(mockRepo, time.Second*2)

	mockID := "bank-123"
	existingAccount := &domain.BankAccount{ID: mockID, BankName: "BCA Lama"}
	input := domain.BankAccountUpdateInput{BankName: "BCA Baru"}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingAccount, nil).Once()
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(b *domain.BankAccount) bool {
			return b.BankName == "BCA Baru"
		})).Return(nil).Once()

		result, err := uc.UpdateAccount(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.Equal(t, "BCA Baru", result.BankName)
		mockRepo.AssertExpectations(t)
	})
}

func TestBankAccountUsecase_DeleteAccount(t *testing.T) {
	mockRepo := new(mocks.BankAccountRepository)
	uc := usecase.NewBankAccountUsecase(mockRepo, time.Second*2)
	mockID := "bank-123"

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()
		err := uc.DeleteAccount(context.Background(), mockID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestBankAccountUsecase_GetGlobalAccounts(t *testing.T) {
	mockRepo := new(mocks.BankAccountRepository)
	uc := usecase.NewBankAccountUsecase(mockRepo, time.Second*2)

	mockAccounts := []domain.BankAccount{{BankName: "BCA Global"}}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetGlobalAccounts", mock.Anything).Return(mockAccounts, nil).Once()

		accounts, err := uc.GetGlobalAccounts(context.Background())

		assert.NoError(t, err)
		assert.Len(t, accounts, 1)
		mockRepo.AssertExpectations(t)
	})
}

func TestBankAccountUsecase_GetUserAccounts(t *testing.T) {
	mockRepo := new(mocks.BankAccountRepository)
	uc := usecase.NewBankAccountUsecase(mockRepo, time.Second*2)

	userID := "user-123"
	mockAccounts := []domain.BankAccount{{BankName: "BCA Sales"}}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByUserID", mock.Anything, userID).Return(mockAccounts, nil).Once()

		accounts, err := uc.GetUserAccounts(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, accounts, 1)
		mockRepo.AssertExpectations(t)
	})
}
