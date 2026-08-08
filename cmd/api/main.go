package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"

	// Sesuaikan module path ini jika berbeda
	"github.com/faridlan/sikon-api/docs"
	"github.com/faridlan/sikon-api/internal/config"
	myHttp "github.com/faridlan/sikon-api/internal/delivery/http"
	"github.com/faridlan/sikon-api/internal/delivery/http/middleware"
	"github.com/faridlan/sikon-api/internal/infrastructure/supabase"
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/usecase"

	// Import docs untuk Swagger (jangan dihapus)
	_ "github.com/faridlan/sikon-api/docs"
)

// @title SIKOn API (Sistem Integrasi Konveksi Online)
// @version 1.0
// @description Ini adalah dokumentasi API untuk MVP ERP SIKOn.
// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Masukkan token dengan format: Bearer {token}
func main() {
	// ==========================================
	// 0. INISIALISASI KONFIGURASI & LOGGER
	// ==========================================
	config.InitLogger()

	err := godotenv.Load()
	if err != nil {
		slog.Warn("File .env tidak ditemukan, menggunakan environment variable dari sistem")
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbURL := os.Getenv("DB_URL") // Contoh: postgres://user:pass@host:5432/dbname?sslmode=disable

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_KEY")
	supabaseBucket := os.Getenv("SUPABASE_BUCKET")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "sikon-secret-key-development"
		slog.Warn("JWT_SECRET tidak diset, menggunakan default development secret key")
	}

	jwtTTLStr := os.Getenv("JWT_TTL_HOURS")
	jwtTTL := 24 * time.Hour
	if jwtTTLStr != "" {
		if ttlHours, err := time.ParseDuration(jwtTTLStr + "h"); err == nil {
			jwtTTL = ttlHours
		}
	}

	// Menjalankan Migrasi Database
	if dbURL != "" {
		config.RunDBMigration(dbURL)
	} else {
		slog.Warn("DB_URL tidak diset, migrasi database dilewati")
	}

	// Inisiasi Koneksi Database
	db := config.InitDB(dbUser, dbPassword, dbHost, dbPort, dbName)

	// Context timeout untuk membatasi lama eksekusi query (Cegah query gantung)
	contextTimeout := 5 * time.Second

	storageService := supabase.NewSupabaseStorage(supabaseURL, supabaseKey, supabaseBucket)

	// ==========================================
	// 1. INISIASI REPOSITORY (Layer Data)
	// ==========================================
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

	// ==========================================
	// 2. INISIASI USECASE (Layer Logika Bisnis)
	// ==========================================
	authUsecase := usecase.NewAuthUsecase(userRepo, jwtSecret, jwtTTL, contextTimeout)
	userUsecase := usecase.NewUserUsecase(userRepo, storageService, txManager, contextTimeout)
	categoryUsecase := usecase.NewCategoryUsecase(categoryRepo, contextTimeout)
	productUsecase := usecase.NewProductUsecase(productRepo, categoryRepo, storageService, txManager, contextTimeout)
	customerUsecase := usecase.NewCustomerUsecase(customerRepo, userRepo, contextTimeout)
	bankAccountUsecase := usecase.NewBankAccountUsecase(bankAccountRepo, contextTimeout)
	dashboardUsecase := usecase.NewDashboardUsecase(dashboardRepo, contextTimeout)
	reportUsecase := usecase.NewReportUsecase(reportRepo, contextTimeout)
	batchPOUsecase := usecase.NewBatchPOUsecase(batchPORepo, contextTimeout)
	uploadUsecase := usecase.NewUploadUsecase(storageService, contextTimeout)
	expenseUsecase := usecase.NewExpenseUsecase(expenseRepo, contextTimeout)

	orderUsecase := usecase.NewOrderUsecase(orderRepo, customerRepo, userRepo, productRepo, batchPORepo, paymentRepo, txManager, contextTimeout)
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo, orderRepo, bankAccountRepo, txManager, batchPORepo, contextTimeout)
	specTemplateUsecase := usecase.NewSpecTemplateUsecase(specTemplateRepo, contextTimeout)

	// ==========================================
	// 3. INISIASI HANDLER (Layer Delivery)
	// ==========================================
	authHandler := myHttp.NewAuthHandler(authUsecase)
	userHandler := myHttp.NewUserHandler(userUsecase)
	categoryHandler := myHttp.NewCategoryHandler(categoryUsecase)
	productHandler := myHttp.NewProductHandler(productUsecase)
	customerHandler := myHttp.NewCustomerHandler(customerUsecase)
	bankAccountHandler := myHttp.NewBankAccountHandler(bankAccountUsecase)
	orderHandler := myHttp.NewOrderHandler(orderUsecase)
	paymentHandler := myHttp.NewPaymentHandler(paymentUsecase)
	specTemplateHandler := myHttp.NewSpecTemplateHandler(specTemplateUsecase)
	dashboardHandler := myHttp.NewDashboardHandler(dashboardUsecase)
	reportHandler := myHttp.NewReportHandler(reportUsecase)
	batchPOHandler := myHttp.NewBatchPOHandler(batchPOUsecase)
	uploadHandler := myHttp.NewUploadHandler(uploadUsecase)
	expenseHandler := myHttp.NewExpenseHandler(expenseUsecase)
	seederHandler := myHttp.NewSeederHandler(db, storageService)

	// ==========================================
	// 4. BUNGKUS KE DALAM STRUCT REGISTRY ROUTER
	// ==========================================
	handlers := myHttp.Handlers{
		AuthHandler:         authHandler,
		UserHandler:         userHandler,
		CategoryHandler:     categoryHandler,
		CustomerHandler:     customerHandler,
		ProductHandler:      productHandler,
		BankAccountHandler:  bankAccountHandler,
		OrderHandler:        orderHandler,
		PaymentHandler:      paymentHandler,
		SpecTemplateHandler: specTemplateHandler,
		DashboardHandler:    dashboardHandler,
		ReportHandler:       reportHandler,
		BatchPOHandler:      batchPOHandler,
		UploadHandler:       uploadHandler,
		ExpenseHandler:      expenseHandler,
		SeederHandler:       seederHandler,
	}

	// ==========================================
	// 5. SETUP FIBER APP
	// ==========================================
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "*"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     frontendURL,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-User-Id, X-Requested-With",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: false,
	}))

	app.Use(requestid.New())
	app.Use(middleware.SlogMiddleware())

	// Setup Swagger
	swaggerHost := os.Getenv("SWAGGER_HOST")
	if swaggerHost != "" {
		docs.SwaggerInfo.Host = swaggerHost
	}
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Daftarkan semua route dari router.go dengan JWT Secret
	myHttp.SetupRoutes(app, handlers, jwtSecret)

	// ==========================================
	// 6. JALANKAN SERVER DENGAN GRACEFUL SHUTDOWN
	// ==========================================
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		slog.Info("🚀 SIKOn API Server is running", slog.String("port", port))
		if err := app.Listen(":" + port); err != nil {
			slog.Error("Server gagal berjalan", slog.String("detail", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit

	slog.Info("Menerima sinyal mati, mematikan server dengan sopan...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		slog.Error("Server dipaksa mati karena timeout atau error", slog.String("detail", err.Error()))
	}

	slog.Info("✅ SIKOn API Server berhasil dimatikan dengan aman. Sampai jumpa!")
}
