package usecase

import (
	"context"
	"errors" // Tambahan import
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type customerUsecase struct {
	customerRepo   domain.CustomerRepository
	contextTimeout time.Duration
}

func NewCustomerUsecase(cr domain.CustomerRepository, timeout time.Duration) domain.CustomerUsecase {
	return &customerUsecase{
		customerRepo:   cr,
		contextTimeout: timeout,
	}
}

func (u *customerUsecase) CreateCustomer(c context.Context, input domain.CustomerCreateInput) (*domain.Customer, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	customer := &domain.Customer{
		Name:      input.Name,
		Phone:     input.Phone,
		Address:   input.Address,
		CreatedBy: input.CreatedBy,
	}

	if err := u.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}

func (u *customerUsecase) GetCustomer(c context.Context, id string) (*domain.Customer, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	customer, err := u.customerRepo.GetByID(ctx, id)
	if err != nil {
		// PENAMBAHAN IF STATEMENT
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Data pelanggan tidak ditemukan")
		}
		return nil, err
	}

	return customer, nil
}

func (u *customerUsecase) ListCustomers(c context.Context, query domain.PaginationQuery) ([]domain.Customer, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	customers, totalItems, err := u.customerRepo.Fetch(ctx, limit, offset)
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

	return customers, meta, nil
}

func (u *customerUsecase) UpdateCustomer(c context.Context, id string, input domain.CustomerUpdateInput) (*domain.Customer, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingCustomer, err := u.customerRepo.GetByID(ctx, id)
	if err != nil {
		// PENAMBAHAN IF STATEMENT
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Data pelanggan tidak ditemukan")
		}
		return nil, err
	}

	if input.Name != "" {
		existingCustomer.Name = input.Name
	}
	if input.Phone != "" {
		existingCustomer.Phone = input.Phone
	}
	if input.Address != "" {
		existingCustomer.Address = input.Address
	}

	if err := u.customerRepo.Update(ctx, existingCustomer); err != nil {
		return nil, err
	}

	return existingCustomer, nil
}

func (u *customerUsecase) DeleteCustomer(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.customerRepo.Delete(ctx, id)
}
