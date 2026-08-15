package middleware

import (
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

// JWTMiddleware memvalidasi token dari header Authorization: Bearer <token>
func JWTMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return utils.SendError(c, fiber.StatusUnauthorized, "Header otorisasi tidak ditemukan")
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return utils.SendError(c, fiber.StatusUnauthorized, "Format token tidak valid. Gunakan: Bearer <token>")
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return utils.SendError(c, fiber.StatusUnauthorized, "Token tidak valid atau telah kadaluwarsa")
		}

		claimsMap, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, "Gagal memproses claims token")
		}

		claims := domain.AuthClaims{
			UserID: claimsMap["user_id"].(string),
			Email:  claimsMap["email"].(string),
			Role:   domain.Role(claimsMap["role"].(string)),
		}

		// Simpan claims ke locals Fiber agar bisa dibaca handler lain
		c.Locals("user", claims)
		c.Locals("userID", claims.UserID)
		c.Locals("userRole", claims.Role)

		return c.Next()
	}
}

// RoleGuard membatasi akses endpoint hanya untuk role tertentu
func RoleGuard(allowedRoles ...domain.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("userRole").(domain.Role)
		if !ok {
			return utils.SendError(c, fiber.StatusForbidden, "Akses ditolak: Identitas pengguna tidak ditemukan")
		}

		// Owner selalu mendapatkan akses penuh (Superuser)
		if userRole == domain.RoleOwner {
			return c.Next()
		}

		if slices.Contains(allowedRoles, userRole) {
			return c.Next()
		}

		return utils.SendError(c, fiber.StatusForbidden, "Akses ditolak: Anda tidak memiliki hak akses untuk fitur ini")
	}
}
