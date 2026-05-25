package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/domain/mocks"
	"github.com/faridlan/sikon-api/internal/usecase"
)

func TestUserUsecase_Register(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	uc := usecase.NewUserUsecase(mockRepo, time.Second*2)

	input := domain.UserRegisterInput{
		Name:     "Budi",
		Email:    "budi@example.com",
		Password: "password123",
		Role:     domain.RoleSales,
	}

	t.Run("Success", func(t *testing.T) {
		// Ekspektasi: Repo Create dipanggil dengan parameter User yang nilainya cocok.
		// Karena password di-hash secara acak, kita pakai mock.MatchedBy untuk mengecek nilai lainnya.
		mockRepo.On("GetByEmail", mock.Anything, input.Email).Return(nil, domain.ErrNotFound).Once() // Pastikan email belum terdaftar
		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.Name == input.Name && u.Email == input.Email && u.Role == input.Role
		})).Return(nil).Once()

		result, err := uc.Register(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		assert.Equal(t, input.Email, result.Email)

		// Verifikasi apakah password benar-benar di-hash menggunakan bcrypt
		errBcrypt := bcrypt.CompareHashAndPassword([]byte(result.Password), []byte(input.Password))
		assert.NoError(t, errBcrypt, "Password harus di-hash dengan benar")

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Invalid Role", func(t *testing.T) {
		invalidInput := input
		invalidInput.Role = "superadmin" // Role yang tidak diizinkan

		mockRepo.On("GetByEmail", mock.Anything, invalidInput.Email).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.Register(context.Background(), invalidInput)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
	})

	t.Run("Error - Repo Failed", func(t *testing.T) {
		repoError := errors.New("database error: email already exists")

		mockRepo.On("GetByEmail", mock.Anything, input.Email).Return(nil, domain.ErrNotFound).Once()
		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(repoError).Once()

		result, err := uc.Register(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, repoError, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_GetProfile(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	uc := usecase.NewUserUsecase(mockRepo, time.Second*2)

	mockID := "user-123"
	mockUser := &domain.User{
		ID:    mockID,
		Name:  "Budi",
		Email: "budi@example.com",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockUser, nil).Once()

		result, err := uc.GetProfile(context.Background(), mockID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, mockUser.Name, result.Name)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetProfile(context.Background(), mockID)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrNotFound, appErr.ErrType)

		mockRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_ListUsers(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	uc := usecase.NewUserUsecase(mockRepo, time.Second*2)

	query := domain.PaginationQuery{Page: 1, Limit: 10}

	mockUsers := []domain.User{
		{ID: "1", Name: "User 1"},
		{ID: "2", Name: "User 2"},
	}
	var totalItems int64 = 15

	t.Run("Success", func(t *testing.T) {
		// Offset dihitung: (Page - 1) * Limit -> (1 - 1) * 10 = 0
		mockRepo.On("Fetch", mock.Anything, 10, 0).Return(mockUsers, totalItems, nil).Once()

		users, meta, err := uc.ListUsers(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, users, 2)

		// Total Pages seharusnya 2 (karena totalItems 15 / limit 10 -> dibulatkan ke atas jadi 2)
		assert.Equal(t, 2, meta.TotalPages)
		assert.Equal(t, int64(15), meta.TotalItems)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repo Failed", func(t *testing.T) {
		repoError := errors.New("db error")
		mockRepo.On("Fetch", mock.Anything, 10, 0).Return(nil, int64(0), repoError).Once()

		users, _, err := uc.ListUsers(context.Background(), query)

		assert.Error(t, err)
		assert.Nil(t, users)
		assert.Equal(t, repoError, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_UpdateUser(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	uc := usecase.NewUserUsecase(mockRepo, time.Second*2)

	mockID := "user-123"
	input := domain.UserUpdateInput{
		Name: "Budi Updated",
		Role: domain.RoleAdmin,
	}

	// Data yang ada di DB sebelum di-update
	existingUser := &domain.User{
		ID:   mockID,
		Name: "Budi Lama",
		Role: domain.RoleSales,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingUser, nil).Once()

		// Ekspektasi saat Update dipanggil, datanya sudah berubah sesuai input
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.Name == input.Name && u.Role == input.Role
		})).Return(nil).Once()

		result, err := uc.UpdateUser(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Budi Updated", result.Name)
		assert.Equal(t, domain.RoleAdmin, result.Role)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - User Not Found", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.UpdateUser(context.Background(), mockID, input)

		assert.Error(t, err)
		assert.Nil(t, result)

		mockRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_DeleteUser(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	uc := usecase.NewUserUsecase(mockRepo, time.Second*2)

	mockID := "user-123"
	existingUser := &domain.User{
		ID:   mockID,
		Name: "Budi Lama",
		Role: domain.RoleSales,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingUser, nil).Once()
		mockRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()

		err := uc.DeleteUser(context.Background(), mockID)

		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})
}
