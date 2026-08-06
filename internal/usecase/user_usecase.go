package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type userUsecase struct {
	userRepo       domain.UserRepository
	storageService domain.StorageService
	txManager      domain.TransactionManager
	contextTimeout time.Duration
}

func NewUserUsecase(ur domain.UserRepository, ss domain.StorageService, tm domain.TransactionManager, timeout time.Duration) domain.UserUsecase {
	return &userUsecase{
		userRepo:       ur,
		storageService: ss,
		txManager:      tm,
		contextTimeout: timeout,
	}
}

func (u *userUsecase) Register(c context.Context, input domain.UserRegisterInput) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	_, err := u.userRepo.GetByEmail(ctx, input.Email)
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

	statusText := input.StatusText
	if statusText == "" {
		statusText = "Online sekarang"
	}

	user := &domain.User{
		Name:       input.Name,
		Email:      input.Email,
		Password:   string(hashedPassword),
		Role:       input.Role,
		ImageURL:   input.ImageURL,
		Phone:      input.Phone,
		StatusText: statusText,
		IsActive:   input.IsActive,
		SortOrder:  input.SortOrder,
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
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "User tidak ditemukan")
		}
		return nil, err
	}

	return user, nil
}

func (u *userUsecase) ListUsers(c context.Context, query domain.PaginationQuery, filter domain.UserFilter) ([]domain.User, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	users, totalItems, err := u.userRepo.Fetch(ctx, limit, offset, filter)
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

// GetPublicSalesList melayani request publik tanpa autentikasi untuk widget WhatsApp Sales
func (u *userUsecase) GetPublicSalesList(c context.Context) ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	users, err := u.userRepo.GetPublicSalesList(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (u *userUsecase) UpdateUser(c context.Context, id string, input domain.UserUpdateInput) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingUser, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
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
	if input.Phone != "" {
		existingUser.Phone = input.Phone
	}
	if input.StatusText != "" {
		existingUser.StatusText = input.StatusText
	}
	if input.IsActive != nil {
		existingUser.IsActive = *input.IsActive
	}
	if input.SortOrder != nil {
		existingUser.SortOrder = *input.SortOrder
	}

	oldImageURL := existingUser.ImageURL
	if input.ImageURL != "" {
		existingUser.ImageURL = input.ImageURL
	}

	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := u.userRepo.Update(txCtx, existingUser); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if input.ImageURL != "" && oldImageURL != "" && oldImageURL != input.ImageURL {
		_ = u.storageService.DeleteFile(ctx, oldImageURL)
	}

	return existingUser, nil
}

func (u *userUsecase) DeleteUser(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingUser, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "User tidak ditemukan")
		}
		return err
	}

	err = u.userRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	if existingUser.ImageURL != "" {
		_ = u.storageService.DeleteFile(ctx, existingUser.ImageURL)
	}

	return nil
}
