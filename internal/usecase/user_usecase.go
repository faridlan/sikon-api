package usecase

import (
	"context"
	"errors" // Tambahan import
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type userUsecase struct {
	userRepo       domain.UserRepository
	contextTimeout time.Duration
}

// NewUserUsecase adalah constructor
func NewUserUsecase(ur domain.UserRepository, timeout time.Duration) domain.UserUsecase {
	return &userUsecase{
		userRepo:       ur,
		contextTimeout: timeout,
	}
}

func (u *userUsecase) Register(c context.Context, input domain.UserRegisterInput) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	_, err := u.userRepo.GetByEmail(ctx, input.Email) // Cek dulu apakah email sudah terdaftar
	if err == nil {
		return nil, domain.NewError(domain.ErrConflict, "Email sudah terdaftar")
	}

	if input.Role != domain.RoleAdmin && input.Role != domain.RoleSales {
		return nil, domain.NewError(domain.ErrBadParamInput, "Role tidak valid")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalServerError, "Gagal memproses password")
	}

	user := &domain.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword),
		Role:     input.Role,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *userUsecase) GetProfile(c context.Context, userID string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		// PENAMBAHAN IF STATEMENT
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "User tidak ditemukan")
		}
		return nil, err
	}

	return user, nil
}

func (u *userUsecase) ListUsers(c context.Context, query domain.PaginationQuery) ([]domain.User, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	users, totalItems, err := u.userRepo.Fetch(ctx, limit, offset)
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

	return users, meta, nil
}

func (u *userUsecase) UpdateUser(c context.Context, id string, input domain.UserUpdateInput) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingUser, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		// PENAMBAHAN IF STATEMENT
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "User tidak ditemukan")
		}
		return nil, err
	}

	if input.Name != "" {
		existingUser.Name = input.Name
	}
	if input.Role != "" {
		existingUser.Role = input.Role
	}

	if err := u.userRepo.Update(ctx, existingUser); err != nil {
		return nil, err
	}

	return existingUser, nil
}

func (u *userUsecase) DeleteUser(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	_, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		// PENAMBAHAN IF STATEMENT
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "User tidak ditemukan")
		}
		return err
	}

	return u.userRepo.Delete(ctx, id)
}
