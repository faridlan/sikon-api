package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type bankAccountUsecase struct {
	bankAccountRepo domain.BankAccountRepository
	contextTimeout  time.Duration
}

func NewBankAccountUsecase(br domain.BankAccountRepository, timeout time.Duration) domain.BankAccountUsecase {
	return &bankAccountUsecase{
		bankAccountRepo: br,
		contextTimeout:  timeout,
	}
}

func (u *bankAccountUsecase) CreateAccount(c context.Context, input domain.BankAccountCreateInput) (*domain.BankAccount, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Mapping dari Input Struct ke Entitas Domain
	account := &domain.BankAccount{
		UserID:        input.UserID,
		BankName:      input.BankName,
		AccountNumber: input.AccountNumber,
		AccountName:   input.AccountName,
	}

	if err := u.bankAccountRepo.Create(ctx, account); err != nil {
		return nil, err
	}

	// Mengembalikan entitas utuh yang sudah terisi ID dan CreatedAt dari database
	return account, nil
}

func (u *bankAccountUsecase) GetAccount(c context.Context, id string) (*domain.BankAccount, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	account, err := u.bankAccountRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Rekening tidak ditemukan")
		}
		return nil, err
	}

	return account, nil
}

func (u *bankAccountUsecase) ListAccounts(c context.Context, query domain.PaginationQuery) ([]domain.BankAccount, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	accounts, totalItems, err := u.bankAccountRepo.Fetch(ctx, limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	return accounts, meta, nil
}

func (u *bankAccountUsecase) UpdateAccount(c context.Context, id string, input domain.BankAccountUpdateInput) (*domain.BankAccount, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingAccount, err := u.bankAccountRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Rekening tidak ditemukan")
		}
		return nil, err
	}

	// Hanya update nilai yang dikirimkan
	if input.BankName != "" {
		existingAccount.BankName = input.BankName
	}
	if input.AccountNumber != "" {
		existingAccount.AccountNumber = input.AccountNumber
	}
	if input.AccountName != "" {
		existingAccount.AccountName = input.AccountName
	}

	if err := u.bankAccountRepo.Update(ctx, existingAccount); err != nil {
		return nil, err
	}

	return existingAccount, nil
}

func (u *bankAccountUsecase) DeleteAccount(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	_, err := u.bankAccountRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Rekening tidak ditemukan")
		}
		return err
	}

	return u.bankAccountRepo.Delete(ctx, id)
}

func (u *bankAccountUsecase) GetGlobalAccounts(c context.Context) ([]domain.BankAccount, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.bankAccountRepo.GetGlobalAccounts(ctx)
}

func (u *bankAccountUsecase) GetUserAccounts(c context.Context, userID string) ([]domain.BankAccount, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.bankAccountRepo.GetByUserID(ctx, userID)
}
