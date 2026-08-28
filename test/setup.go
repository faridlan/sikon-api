package tests

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
		&postgres.WorkerModel{},
		&postgres.PayrollModel{},
		&postgres.WorkLogModel{},
		&postgres.AttendanceModel{},
	)

	// Pastikan ada test user yang valid di database untuk endpoint yang menggunakan CreatedBy dari JWT
	defaultUserID := normalizeTestUserID("test-user")
	db.FirstOrCreate(&postgres.UserModel{ID: defaultUserID}, postgres.UserModel{
		ID:       defaultUserID,
		Name:     "Test User",
		Email:    "test-user@sikon.com",
		Password: "test-password",
		Role:     string(domain.RoleOwner),
		IsActive: true,
	})

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
	workerRepo := postgres.NewWorkerRepository(db)
	workLogRepo := postgres.NewWorkLogRepository(db)
	payrollRepo := postgres.NewPayrollRepository(db)
	attendanceRepo := postgres.NewAttendanceRepository(db)

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
	workerUsecase := usecase.NewWorkerUsecase(workerRepo, timeout)
	workLogUsecase := usecase.NewWorkLogUsecase(workLogRepo, workerRepo, orderRepo, batchPORepo, timeout)
	payrollUsecase := usecase.NewPayrollUsecase(payrollRepo, expenseRepo, timeout)
	attendanceUsecase := usecase.NewAttendanceUsecase(attendanceRepo, workerRepo, timeout)

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
		WorkerHandler:       myHttp.NewWorkerHandler(workerUsecase),
		WorkLogHandler:      myHttp.NewWorkLogHandler(workLogUsecase),
		PayrollHandler:      myHttp.NewPayrollHandler(payrollUsecase),
		AttendanceHandler:   myHttp.NewAttendanceHandler(attendanceUsecase),
	}

	app := fiber.New()
	myHttp.SetupRoutes(app, handlers, jwtSecret)

	return app, db
}

// GenerateTestJWTToken adalah helper untuk membuat token Authorization valid saat tes integrasi
func normalizeTestUserID(userID string) string {
	if userID == "" {
		return uuid.NewString()
	}

	if _, err := uuid.Parse(userID); err == nil {
		return userID
	}

	return uuid.NewMD5(uuid.NameSpaceOID, []byte(userID)).String()
}

func GenerateTestJWTToken(userID string, email string, role domain.Role) string {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = TestJWTSecret
	}

	normalizedUserID := normalizeTestUserID(userID)

	claims := jwt.MapClaims{
		"user_id": normalizedUserID,
		"email":   email,
		"role":    string(role),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecret))
	return tokenString
}

// AuthenticatedRequest membuat request HTTP dengan header Authorization Bearer JWT
func AuthenticatedRequest(method string, url string, body io.Reader, userID string, email string, role domain.Role) *http.Request {
	token := GenerateTestJWTToken(userID, email, role)
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// ClearTables mengosongkan seluruh isi tabel sebelum/sesudah tiap test case berjalan
func ClearTables(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE work_logs RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE payrolls RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE workers RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE expenses RESTART IDENTITY CASCADE;")
	db.Exec("TRUNCATE TABLE expense_categories RESTART IDENTITY CASCADE;")
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
	db.Exec("TRUNCATE TABLE attendances RESTART IDENTITY CASCADE;")

	// Re-create default test user used by generic AuthenticatedRequest calls.
	defaultUserID := normalizeTestUserID("test-user")
	db.FirstOrCreate(&postgres.UserModel{ID: defaultUserID}, postgres.UserModel{
		ID:         defaultUserID,
		Name:       "Test User",
		Email:      "test-user@sikon.com",
		Password:   "test-password",
		Role:       string(domain.RoleOwner),
		IsActive:   true,
		StatusText: "Online sekarang",
	})
}
