package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type customerUsecase struct {
	customerRepo   domain.CustomerRepository
	userRepo       domain.UserRepository
	contextTimeout time.Duration
}

func NewCustomerUsecase(cr domain.CustomerRepository, ur domain.UserRepository, timeout time.Duration) domain.CustomerUsecase {
	return &customerUsecase{
		customerRepo:   cr,
		userRepo:       ur,
		contextTimeout: timeout,
	}
}

func (u *customerUsecase) CreateCustomer(c context.Context, input domain.CustomerCreateInput) (*domain.Customer, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// --- TAMBAHAN LOGIKA BISNIS: Validasi Role Sales ---
	if input.SalesID != "" {
		// Cari user berdasarkan SalesID
		salesUser, err := u.userRepo.GetByID(ctx, input.SalesID)
		if err != nil {
			return nil, domain.NewError(domain.ErrNotFound, "Sales ID tidak ditemukan di sistem")
		}

		// Pastikan user tersebut role-nya adalah Sales
		if salesUser.Role != domain.RoleSales {
			return nil, domain.NewError(domain.ErrBadParamInput, "User yang ditugaskan bukan seorang Sales")
		}
	}
	// ---------------------------------------------------

	customer := &domain.Customer{
		Name:      input.Name,
		Phone:     input.Phone,
		Address:   input.Address,
		CreatedBy: input.CreatedBy,
		SalesID:   input.SalesID, // Simpan SalesID yang sudah tervalidasi
	}

	if err := u.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}

func (u *customerUsecase) GetCustomer(c context.Context, id string, operatorID string, operatorRole domain.Role) (*domain.Customer, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	customer, err := u.customerRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Data pelanggan tidak ditemukan")
		}
		return nil, err
	}

	// --- TAMBAHAN LOGIKA BISNIS: Keamanan Hak Akses ---
	// Jika yang request adalah Sales, pastikan Customer ini adalah milik dia
	if operatorRole == domain.RoleSales && customer.SalesID != operatorID {
		return nil, domain.NewError(domain.ErrForbidden, "Akses ditolak. Anda tidak memiliki hak untuk melihat pelanggan ini.")
	}
	// (Jika operatorRole == RoleAdmin, logika di atas dilewati, sehingga Admin bisa melihat semuanya)
	// --------------------------------------------------

	return customer, nil
}

func (u *customerUsecase) ListCustomers(c context.Context, query domain.PaginationQuery, requestedSalesID, operatorID string, operatorRole domain.Role) ([]domain.Customer, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	// --- TAMBAHAN LOGIKA BISNIS: Penentuan Filter ---
	var filterSalesID string
	if operatorRole == domain.RoleSales {
		// Paksa filter hanya mengambil data milik sales ini saja
		filterSalesID = operatorID
	} else if requestedSalesID != "" {
		// Jika ada request specific sales ID, gunakan itu
		filterSalesID = requestedSalesID
	}
	// ------------------------------------------------

	// Panggil Fetch dari Repo dengan menambahkan filterSalesID
	customers, totalItems, err := u.customerRepo.Fetch(ctx, limit, offset, filterSalesID)
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
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Data pelanggan tidak ditemukan")
		}
		return nil, err
	}

	// --- TAMBAHAN LOGIKA BISNIS: Update Sales ID ---
	// Cek apakah ada request perubahan SalesID dan ID-nya beda dari yang lama
	if input.SalesID != "" && input.SalesID != existingCustomer.SalesID {
		salesUser, err := u.userRepo.GetByID(ctx, input.SalesID)
		if err != nil {
			return nil, domain.NewError(domain.ErrNotFound, "Sales ID tidak ditemukan di sistem")
		}
		if salesUser.Role != domain.RoleSales {
			return nil, domain.NewError(domain.ErrBadParamInput, "User yang ditugaskan bukan seorang Sales")
		}
		// Timpa data SalesID lama dengan yang baru
		existingCustomer.SalesID = input.SalesID
	}
	// -----------------------------------------------

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

	_, err := u.customerRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Data pelanggan tidak ditemukan")
		}
		return err
	}

	return u.customerRepo.Delete(ctx, id)
}
