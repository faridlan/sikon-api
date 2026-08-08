package tests

import (
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/faridlan/sikon-api/internal/config"
	myHttp "github.com/faridlan/sikon-api/internal/delivery/http"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/infrastructure/supabase"
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/usecase"
)

const TestJWTSecret = "test-jwt-secret-key-sikon"

// SetupTestApp menginisialisasi DB dan Fiber App khusus untuk Integration Test
func SetupTestApp() (*fiber.App, *gorm.DB) {
	// 1. Membaca .env.test
	err := godotenv.Overload("../../.env.test")
	if err != nil {
		slog.Warn("Peringatan: Gagal meload .env.test, menggunakan environment variables dari OS/CI-CD")
	}

	// 2. Mengambil kredensial dari .env.test menggunakan os.Getenv
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")
	supabaseBucket := os.Getenv("SUPABASE_BUCKET")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = TestJWTSecret
	}

	slog.Info("INTEGRATION TEST CONFIG",
		slog.String("bucket_detected", supabaseBucket),
		slog.String("url_detected", supabaseURL),
	)

	if supabaseBucket == "" {
		panic("SUPABASE_BUCKET kosong! Pastikan .env.test terbaca dengan benar.")
	}

	// 3. Inisiasi DB Test
	db := config.InitDB(dbUser, dbPassword, dbHost, dbPort, dbName)

	db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)

	// Jalankan AutoMigrate agar seluruh tabel terbuat secara dinamis di DB test
	db.AutoMigrate(
		&postgres.CategoryModel{},
		&postgres.ProductModel{},
		&postgres.ProductImageModel{},
		&postgres.ProductFabricModel{},
		&postgres.FabricColorModel{},
		&postgres.WholesalePriceModel{},
		&postgres.DesignerModel{},
		&postgres.ProductModelViewModel{},
		&postgres.UserModel{},
		&postgres.CustomerModel{},
		&postgres.BankAccountModel{},
		&postgres.OrderModel{},
		&postgres.OrderItemModel{},
		&postgres.PaymentModel{},
		&postgres.SpecTemplateModel{},
		&postgres.BatchPOModel{},
		&postgres.ExpenseCategoryModel{},
		&postgres.ExpenseModel{},
	)

	// Durasi timeout untuk transaksi integrasi
	timeout := 30 * time.Second

	storageService := supabase.NewSupabaseStorage(supabaseURL, supabaseKey, supabaseBucket)

	// 1. Repositories
	userRepo := postgres.NewUserRepository(db)
	categoryRepo := postgres.NewCategoryRepository(db)
	productRepo := postgres.NewProductRepository(db)
	customerRepo := postgres.NewCustomerRepository(db)
	bankAccountRepo := postgres.NewBankAccountRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	paymentRepo := postgres.NewPaymentRepository(db)
	specTemplateRepo := postgres.NewSpecTemplateRepository(db)
	txManager := postgres.NewTransactionManager(db)
	dashboardRepo := postgres.NewDashboardRepository(db)
	reportRepo := postgres.NewReportRepository(db)
	batchPORepo := postgres.NewBatchPORepository(db)
	expenseRepo := postgres.NewExpenseRepository(db)

	// 2. Usecases
	authUsecase := usecase.NewAuthUsecase(userRepo, jwtSecret, 24*time.Hour, timeout)
	userUsecase := usecase.NewUserUsecase(userRepo, storageService, txManager, timeout)
	categoryUsecase := usecase.NewCategoryUsecase(categoryRepo, timeout)
	productUsecase := usecase.NewProductUsecase(productRepo, categoryRepo, storageService, txManager, timeout)
	customerUsecase := usecase.NewCustomerUsecase(customerRepo, userRepo, timeout)
	bankAccountUsecase := usecase.NewBankAccountUsecase(bankAccountRepo, timeout)
	orderUsecase := usecase.NewOrderUsecase(orderRepo, customerRepo, userRepo, productRepo, batchPORepo, paymentRepo, txManager, timeout)
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo, orderRepo, bankAccountRepo, txManager, batchPORepo, timeout)
	specTemplateUsecase := usecase.NewSpecTemplateUsecase(specTemplateRepo, timeout)
	dashboardUsecase := usecase.NewDashboardUsecase(dashboardRepo, timeout)
	reportUsecase := usecase.NewReportUsecase(reportRepo, timeout)
	batchPOUsecase := usecase.NewBatchPOUsecase(batchPORepo, timeout)
	uploadUsecase := usecase.NewUploadUsecase(storageService, timeout)
	expenseUsecase := usecase.NewExpenseUsecase(expenseRepo, timeout)

	// 3. Setup Fiber Handlers
	handlers := myHttp.Handlers{
		AuthHandler:         myHttp.NewAuthHandler(authUsecase),
		UserHandler:         myHttp.NewUserHandler(userUsecase),
		CategoryHandler:     myHttp.NewCategoryHandler(categoryUsecase),
		ProductHandler:      myHttp.NewProductHandler(productUsecase),
		CustomerHandler:     myHttp.NewCustomerHandler(customerUsecase),
		BankAccountHandler:  myHttp.NewBankAccountHandler(bankAccountUsecase),
		OrderHandler:        myHttp.NewOrderHandler(orderUsecase),
		PaymentHandler:      myHttp.NewPaymentHandler(paymentUsecase),
		SpecTemplateHandler: myHttp.NewSpecTemplateHandler(specTemplateUsecase),
		DashboardHandler:    myHttp.NewDashboardHandler(dashboardUsecase),
		ReportHandler:       myHttp.NewReportHandler(reportUsecase),
		BatchPOHandler:      myHttp.NewBatchPOHandler(batchPOUsecase),
		UploadHandler:       myHttp.NewUploadHandler(uploadUsecase),
		ExpenseHandler:      myHttp.NewExpenseHandler(expenseUsecase),
		SeederHandler:       myHttp.NewSeederHandler(db, storageService),
	}

	app := fiber.New()
	myHttp.SetupRoutes(app, handlers, jwtSecret)

	return app, db
}

// GenerateTestJWTToken adalah helper untuk membuat token Authorization valid saat tes integrasi
func GenerateTestJWTToken(userID string, email string, role domain.Role) string {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = TestJWTSecret
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    string(role),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecret))
	return tokenString
}

// ClearTables mengosongkan seluruh isi tabel sebelum/sesudah tiap test case berjalan
func ClearTables(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE categories RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE products RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE product_images RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE product_fabrics RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE fabric_colors RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE wholesale_prices RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE product_models RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE product_model_views RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE customers RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE bank_accounts RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE orders RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE order_items RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE payments RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE spec_templates RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE batch_pos RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE expense_categories RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE expenses RESTART IDENTITY CASCADE;")
}
