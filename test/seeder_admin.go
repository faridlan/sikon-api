package tests

import (
	"log/slog"
	"os"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedInitialAdmin(db *gorm.DB) {
	name := os.Getenv("SEED_ADMIN_NAME")
	email := os.Getenv("SEED_ADMIN_EMAIL")
	password := os.Getenv("SEED_ADMIN_PASSWORD")

	if name == "" {
		name = "Owner Utama"
	}
	if email == "" {
		email = "owner@sikon.com"
	}
	if password == "" {
		password = "AdminPassword123!"
	}

	// 1. Cek apakah user admin/owner dengan email tersebut sudah ada
	var existingUser postgres.UserModel
	err := db.Where("email = ? AND deleted_at IS NULL", email).First(&existingUser).Error

	if err == nil {
		slog.Info("⚠️ Initial admin user sudah ada di database, skipping seed", slog.String("email", email))
		return
	}

	// 👈 UBAH BARIS 37 DI SINI: Langsung cek `err != gorm.ErrRecordNotFound`
	if err != gorm.ErrRecordNotFound {
		slog.Error("❌ Gagal memeriksa data admin di database", slog.String("detail", err.Error()))
		return
	}

	// 2. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("❌ Gagal melakukan hash password admin", slog.String("detail", err.Error()))
		return
	}

	// 3. Insert Admin / Owner Baru
	adminModel := postgres.UserModel{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
		Role:     string(domain.RoleOwner),
	}

	if err := db.Create(&adminModel).Error; err != nil {
		slog.Error("❌ Gagal menyimpan admin user ke database", slog.String("detail", err.Error()))
		return
	}

	slog.Info("✅ BERHASIL SEEDING INITIAL ADMIN / OWNER USER!",
		slog.String("name", name),
		slog.String("email", email),
		slog.String("role", string(domain.RoleOwner)),
	)
}
