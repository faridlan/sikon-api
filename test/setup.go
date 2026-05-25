package tests

import (
	"log/slog"
	"os" // Jangan lupa import os
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/faridlan/sikon-api/internal/config"
	myHttp "github.com/faridlan/sikon-api/internal/delivery/http"
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/usecase"
)

// SetupTestApp menginisialisasi DB dan Fiber App khusus untuk Test
func SetupTestApp() (*fiber.App, *gorm.DB) {
	// 1. Membaca .env.test
	err := godotenv.Load("../../.env.test")
	if err != nil {
		slog.Warn("Peringatan: Gagal meload .env.test, menggunakan environment variables dari OS/CI-CD")
	}

	// 2. Mengambil kredensial dari .env.test menggunakan os.Getenv
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	// 3. Inisiasi DB Test (Masukkan variabel di atas, JANGAN di-hardcode)
	db := config.InitDB(dbUser, dbPassword, dbHost, dbPort, dbName)

	// Jalankan AutoMigrate agar tabel selalu terbuat di DB test
	db.AutoMigrate(
		&postgres.CategoryModel{},
		&postgres.ProductModel{},
		&postgres.UserModel{},
		&postgres.CustomerModel{},
		&postgres.BankAccountModel{},
		&postgres.OrderModel{},
		&postgres.OrderItemModel{},
		&postgres.PaymentModel{},
	)

	timeout := 5 * time.Second

	// 1. Repo
	categoryRepo := postgres.NewCategoryRepository(db)
	productRepo := postgres.NewProductRepository(db) // <-- Tambahkan ini
	userRepo := postgres.NewUserRepository(db)
	customerRepo := postgres.NewCustomerRepository(db)
	bankAccountRepo := postgres.NewBankAccountRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	paymentRepo := postgres.NewPaymentRepository(db)

	// 2. Usecase (Perhatikan bahwa productUsecase juga butuh categoryRepo)
	categoryUsecase := usecase.NewCategoryUsecase(categoryRepo, timeout)
	productUsecase := usecase.NewProductUsecase(productRepo, categoryRepo, timeout) // <-- Tambahkan ini
	userUsecase := usecase.NewUserUsecase(userRepo, timeout)
	customerUsecase := usecase.NewCustomerUsecase(customerRepo, timeout)
	bankAccountUsecase := usecase.NewBankAccountUsecase(bankAccountRepo, timeout)
	orderUsecase := usecase.NewOrderUsecase(orderRepo, customerRepo, userRepo, productRepo, timeout)
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo, orderRepo, bankAccountRepo, timeout)

	// 3. Masukkan ke struct Handlers
	handlers := myHttp.Handlers{
		CategoryHandler:    myHttp.NewCategoryHandler(categoryUsecase),
		ProductHandler:     myHttp.NewProductHandler(productUsecase), // <-- Tambahkan ini
		UserHandler:        myHttp.NewUserHandler(userUsecase),
		CustomerHandler:    myHttp.NewCustomerHandler(customerUsecase),
		BankAccountHandler: myHttp.NewBankAccountHandler(bankAccountUsecase),
		OrderHandler:       myHttp.NewOrderHandler(orderUsecase),
		PaymentHandler:     myHttp.NewPaymentHandler(paymentUsecase),
	}

	app := fiber.New()
	myHttp.SetupRoutes(app, handlers)

	return app, db
}

// ClearTables adalah fungsi ajaib untuk mengosongkan semua isi tabel sebelum tiap test berjalan
func ClearTables(db *gorm.DB) {
	// Hapus product terlebih dahulu (jika ada relasi) lalu categories
	db.Exec("TRUNCATE TABLE categories RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE products RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE customers RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE bank_accounts RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE orders RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE order_items RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE payments RESTART IDENTITY CASCADE;")
}
