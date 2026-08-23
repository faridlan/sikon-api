package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type authUsecase struct {
	userRepo       domain.UserRepository
	jwtSecret      string
	jwtTTL         time.Duration
	contextTimeout time.Duration
}

func NewAuthUsecase(ur domain.UserRepository, jwtSecret string, jwtTTL time.Duration, timeout time.Duration) domain.AuthUsecase {
	return &authUsecase{
		userRepo:       ur,
		jwtSecret:      jwtSecret,
		jwtTTL:         jwtTTL,
		contextTimeout: timeout,
	}
}

func (u *authUsecase) Login(c context.Context, input domain.LoginInput) (*domain.AuthResponse, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	user, err := u.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrUnauthorized, "Email atau password tidak valid")
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, domain.NewError(domain.ErrForbidden, "Akun Anda telah dinonaktifkan, silakan hubungi Owner")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		return nil, domain.NewError(domain.ErrUnauthorized, "Email atau password tidak valid")
	}

	expiresAt := time.Now().Add(u.jwtTTL)
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    string(user.Role),
		"exp":     expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(u.jwtSecret))
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalServerError, "Gagal membuat token autentikasi")
	}

	return &domain.AuthResponse{
		Token:     signedToken,
		ExpiresAt: expiresAt,
		User:      *user,
	}, nil
}
