package domain

import (
	"context"
	"time"
)

type BankAccount struct {
	ID            string
	UserID        *string
	BankName      string
	AccountNumber string
	AccountName   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// BankAccountRepository: Kontrak untuk Database
type BankAccountRepository interface {
	Create(ctx context.Context, account *BankAccount) error
	GetByID(ctx context.Context, id string) (*BankAccount, error)
	Fetch(ctx context.Context, limit, offset int) ([]BankAccount, int64, error)
	Update(ctx context.Context, account *BankAccount) error
	Delete(ctx context.Context, id string) error

	// Custom Query
	GetByUserID(ctx context.Context, userID string) ([]BankAccount, error)
	GetGlobalAccounts(ctx context.Context) ([]BankAccount, error)
}

// BankAccountUsecase: Kontrak untuk Bisnis Logika
type BankAccountUsecase interface {
	CreateAccount(ctx context.Context, account *BankAccount) error
	GetAccount(ctx context.Context, id string) (*BankAccount, error)
	ListAccounts(c context.Context, query PaginationQuery) ([]BankAccount, PaginationMeta, error)
	UpdateAccount(ctx context.Context, account *BankAccount) error
	DeleteAccount(ctx context.Context, id string) error
}
