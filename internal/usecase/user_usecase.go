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
	storageService domain.StorageService
	txManager      domain.TransactionManager
	contextTimeout time.Duration
}

// NewUserUsecase adalah constructor
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
		ImageURL: input.ImageURL,
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

	oldImageURL := existingUser.ImageURL

	if input.ImageURL != "" {
		existingUser.ImageURL = input.ImageURL
	}

	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Update user di dalam transaksi
		if err := u.userRepo.Update(txCtx, existingUser); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if input.ImageURL != "" && oldImageURL != "" && oldImageURL != input.ImageURL {
		// Hapus file lama dari storage
		if err := u.storageService.DeleteFile(ctx, oldImageURL); err != nil {
			// Opsional: Log error di sini agar tidak menggagalkan response user jika transaksi DB sudah sukses
			return nil, domain.NewError(domain.ErrInternalServerError, "Gagal menghapus file lama dari storage")
		}
	}

	return existingUser, nil
}

func (u *userUsecase) DeleteUser(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Tangkap data user ke variabel existingUser
	existingUser, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "User tidak ditemukan")
		}
		return err
	}

	// 2. Hapus data user dari database terlebih dahulu
	err = u.userRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// 3. Cek apakah user memiliki file gambar, jika ada baru hapus dari storage
	if existingUser.ImageURL != "" {
		// Gunakan existingUser.ImageURL, BUKAN id user
		if err := u.storageService.DeleteFile(ctx, existingUser.ImageURL); err != nil {
			// Opsional: Kamu bisa memilih untuk mengembalikan error, atau hanya me-log error ini
			// agar user tetap terhapus walaupun file lamanya gagal dibersihkan.
			return domain.NewError(domain.ErrInternalServerError, "Gagal menghapus file gambar dari storage")
		}
	}

	return nil
}
