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

func TestAuthUsecase_Login(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	jwtSecret := "super-secret-key"
	jwtTTL := time.Hour
	timeout := time.Second * 2

	uc := usecase.NewAuthUsecase(mockRepo, jwtSecret, jwtTTL, timeout)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	mockUser := &domain.User{
		ID:       "user-123",
		Name:     "Owner SIKOn",
		Email:    "owner@sikon.com",
		Password: string(hashedPassword),
		Role:     domain.RoleOwner,
		IsActive: true,
	}

	input := domain.LoginInput{
		Email:    "owner@sikon.com",
		Password: "password123",
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("GetByEmail", mock.Anything, input.Email).Return(mockUser, nil).Once()

		result, err := uc.Login(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Token)
		assert.Equal(t, mockUser.Email, result.User.Email)
		assert.Equal(t, mockUser.Role, result.User.Role)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Email Not Found", func(t *testing.T) {
		invalidInput := input
		invalidInput.Email = "notfound@sikon.com"

		mockRepo.On("GetByEmail", mock.Anything, invalidInput.Email).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.Login(context.Background(), invalidInput)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrUnauthorized, appErr.ErrType)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Wrong Password", func(t *testing.T) {
		invalidInput := input
		invalidInput.Password = "wrongpassword"

		mockRepo.On("GetByEmail", mock.Anything, invalidInput.Email).Return(mockUser, nil).Once()

		result, err := uc.Login(context.Background(), invalidInput)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrUnauthorized, appErr.ErrType)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Inactive Account", func(t *testing.T) {
		inactiveUser := *mockUser
		inactiveUser.IsActive = false

		mockRepo.On("GetByEmail", mock.Anything, input.Email).Return(&inactiveUser, nil).Once()

		result, err := uc.Login(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrForbidden, appErr.ErrType)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repo Internal Error", func(t *testing.T) {
		dbErr := errors.New("database connection failed")

		mockRepo.On("GetByEmail", mock.Anything, input.Email).Return(nil, dbErr).Once()

		result, err := uc.Login(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, dbErr, err)

		mockRepo.AssertExpectations(t)
	})
}
