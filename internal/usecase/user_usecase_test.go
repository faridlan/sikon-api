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
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewUserUsecase(mockRepo, mockStorageService, mockTxManager, time.Second*2)

	input := domain.UserRegisterInput{
		Name:       "Budi",
		Email:      "budi@example.com",
		Password:   "password123",
		Role:       domain.RoleSales,
		Phone:      "6281200000001",
		StatusText: "Online sekarang",
		IsActive:   true,
		SortOrder:  1,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByEmail", mock.Anything, input.Email).Return(nil, domain.ErrNotFound).Once()
		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.Name == input.Name && u.Email == input.Email && u.Role == input.Role && u.Phone == input.Phone
		})).Return(nil).Once()

		result, err := uc.Register(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, input.Name, result.Name)
		assert.Equal(t, input.Email, result.Email)
		assert.Equal(t, input.Phone, result.Phone)
		assert.Equal(t, input.StatusText, result.StatusText)

		errBcrypt := bcrypt.CompareHashAndPassword([]byte(result.Password), []byte(input.Password))
		assert.NoError(t, errBcrypt, "Password harus di-hash dengan benar")

		mockRepo.AssertExpectations(t)
	})

	t.Run("Success With Image URL", func(t *testing.T) {
		inputWithImage := input
		inputWithImage.ImageURL = "https://example.com/image.jpg"

		mockRepo.On("GetByEmail", mock.Anything, inputWithImage.Email).Return(nil, domain.ErrNotFound).Once()
		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.Name == inputWithImage.Name && u.ImageURL == inputWithImage.ImageURL
		})).Return(nil).Once()

		result, err := uc.Register(context.Background(), inputWithImage)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, inputWithImage.ImageURL, result.ImageURL)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Invalid Role", func(t *testing.T) {
		invalidInput := input
		invalidInput.Role = "superadmin"

		mockRepo.On("GetByEmail", mock.Anything, invalidInput.Email).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.Register(context.Background(), invalidInput)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)

		mockRepo.AssertExpectations(t)
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
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewUserUsecase(mockRepo, mockStorageService, mockTxManager, time.Second*2)

	mockID := "user-123"
	mockUser := &domain.User{
		ID:         mockID,
		Name:       "Budi",
		Email:      "budi@example.com",
		ImageURL:   "https://example.com/image.jpg",
		Role:       domain.RoleSales,
		Phone:      "6281200000001",
		StatusText: "Online sekarang",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(mockUser, nil).Once()

		result, err := uc.GetProfile(context.Background(), mockID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, mockUser.Name, result.Name)
		assert.Equal(t, mockUser.Phone, result.Phone)

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
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewUserUsecase(mockRepo, mockStorageService, mockTxManager, time.Second*2)

	query := domain.PaginationQuery{Page: 1, Limit: 10}

	mockUsers := []domain.User{
		{ID: "1", Name: "User 1", Role: domain.RoleSales, ImageURL: "https://example.com/image1.jpg"},
		{ID: "2", Name: "User 2", Role: domain.RoleAccounting, ImageURL: "https://example.com/image2.jpg"},
	}
	var totalItems int64 = 15

	t.Run("Success_TanpaFilter", func(t *testing.T) {
		emptyFilter := domain.UserFilter{}

		mockRepo.On("Fetch", mock.Anything, 10, 0, emptyFilter).Return(mockUsers, totalItems, nil).Once()

		users, meta, err := uc.ListUsers(context.Background(), query, emptyFilter)

		assert.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, 2, meta.TotalPages)
		assert.Equal(t, int64(15), meta.TotalItems)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Success_DenganFilterRole", func(t *testing.T) {
		activeFilter := domain.UserFilter{Role: "sales"}

		mockSalesUsers := []domain.User{
			{ID: "1", Name: "User 1", Role: domain.RoleSales},
		}

		mockRepo.On("Fetch", mock.Anything, 10, 0, activeFilter).Return(mockSalesUsers, int64(1), nil).Once()

		users, meta, err := uc.ListUsers(context.Background(), query, activeFilter)

		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, int64(1), meta.TotalItems)

		mockRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_GetPublicSalesList(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewUserUsecase(mockRepo, mockStorageService, mockTxManager, time.Second*2)

	mockSalesList := []domain.User{
		{ID: "sales-1", Name: "Rina Pratiwi", Role: domain.RoleSales, Phone: "6281200000001", StatusText: "Online sekarang", IsActive: true, SortOrder: 1},
		{ID: "sales-2", Name: "Dimas Aditya", Role: domain.RoleSales, Phone: "6281200000002", StatusText: "Online sekarang", IsActive: true, SortOrder: 2},
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetPublicSalesList", mock.Anything).Return(mockSalesList, nil).Once()

		results, err := uc.GetPublicSalesList(context.Background())

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "Rina Pratiwi", results[0].Name)
		assert.Equal(t, "6281200000001", results[0].Phone)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error_From_Repository", func(t *testing.T) {
		expectedErr := errors.New("database connection error")
		mockRepo.On("GetPublicSalesList", mock.Anything).Return(nil, expectedErr).Once()

		results, err := uc.GetPublicSalesList(context.Background())

		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Equal(t, expectedErr, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_UpdateUser(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewUserUsecase(mockRepo, mockStorageService, mockTxManager, time.Second*2)

	mockID := "user-123"
	isActiveVal := true
	sortOrderVal := 1

	input := domain.UserUpdateInput{
		Name:       "Budi Updated",
		Role:       domain.RoleOwner,
		ImageURL:   "https://example.com/new_image.jpg",
		Phone:      "6281299998888",
		StatusText: "Balas dalam 1 jam",
		IsActive:   &isActiveVal,
		SortOrder:  &sortOrderVal,
	}

	existingUser := &domain.User{
		ID:       mockID,
		Name:     "Budi Lama",
		Role:     domain.RoleSales,
		ImageURL: "https://example.com/old_image.jpg",
		Phone:    "6281200000001",
	}

	setupTxMock := func(mockTx *mocks.TransactionManager) {
		mockTx.On("RunInTransaction", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(context.Context) error)
				_ = fn(args.Get(0).(context.Context))
			}).
			Return(nil)
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingUser, nil).Once()

		setupTxMock(mockTxManager)
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.Name == input.Name && u.Phone == input.Phone && u.StatusText == input.StatusText && u.Role == input.Role
		})).Return(nil).Once()

		mockStorageService.On("DeleteFile", mock.Anything, existingUser.ImageURL).Return(nil).Once()

		result, err := uc.UpdateUser(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Budi Updated", result.Name)
		assert.Equal(t, domain.RoleOwner, result.Role)
		assert.Equal(t, "6281299998888", result.Phone)
		assert.Equal(t, "Balas dalam 1 jam", result.StatusText)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Invalid Role", func(t *testing.T) {
		invalidRoleInput := input
		invalidRoleInput.Role = "invalid_role"

		mockRepo.On("GetByID", mock.Anything, mockID).Return(existingUser, nil).Once()

		result, err := uc.UpdateUser(context.Background(), mockID, invalidRoleInput)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)

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
	mockStorageService := new(mocks.StorageService)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewUserUsecase(mockRepo, mockStorageService, mockTxManager, time.Second*2)

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
