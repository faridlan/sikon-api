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

func (u *bankAccountUsecase) CreateAccount(c context.Context, account *domain.BankAccount) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if account.BankName == "" || account.AccountNumber == "" || account.AccountName == "" {
		return domain.NewError(domain.ErrBadParamInput, "Data bank, nomor rekening, dan nama pemilik harus diisi")
	}

	return u.bankAccountRepo.Create(ctx, account)
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

func (u *bankAccountUsecase) UpdateAccount(c context.Context, account *domain.BankAccount) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingAccount, err := u.bankAccountRepo.GetByID(ctx, account.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Rekening tidak ditemukan")
		}
		return err
	}

	if account.BankName != "" {
		existingAccount.BankName = account.BankName
	}
	if account.AccountNumber != "" {
		existingAccount.AccountNumber = account.AccountNumber
	}
	if account.AccountName != "" {
		existingAccount.AccountName = account.AccountName
	}

	// Catatan: Biasanya UserID (pemilik rekening) tidak diizinkan untuk diubah.
	// Jika ingin merubah status dari rekening Sales menjadi rekening Global, harus diatur secara eksplisit.

	return u.bankAccountRepo.Update(ctx, existingAccount)
}

func (u *bankAccountUsecase) DeleteAccount(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.bankAccountRepo.Delete(ctx, id)
}

// Custom Method: Mengambil rekening global perusahaan (sangat berguna saat customer mau transfer DP/Lunas ke perusahaan)
func (u *bankAccountUsecase) GetGlobalAccounts(c context.Context) ([]domain.BankAccount, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Panggil repo GetGlobalAccounts (yang sudah kita buat akan mengirimkan userID kosong/nil)
	return u.bankAccountRepo.GetGlobalAccounts(ctx)
}

// Custom Method: Mengambil list rekening milik spesifik user (Sales)
func (u *bankAccountUsecase) GetUserAccounts(c context.Context, userID string) ([]domain.BankAccount, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.bankAccountRepo.GetByUserID(ctx, userID)
}
