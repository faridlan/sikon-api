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
	// Asumsi: Anda membuat implementasi GORM di folder repository/postgres
	userRepo := postgres.NewUserRepository(db)
	categoryRepo := postgres.NewCategoryRepository(db)
	productRepo := postgres.NewProductRepository(db)
	customerRepo := postgres.NewCustomerRepository(db)
	bankAccountRepo := postgres.NewBankAccountRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	paymentRepo := postgres.NewPaymentRepository(db)
	specTemplateRepo := postgres.NewSpecTemplateRepository(db)
	txManager := postgres.NewTransactionManager(db)
	dashboardRepo := postgres.NewDashboardRepository(db) // Inisialisasi repository dashboard
	reportRepo := postgres.NewReportRepository(db)
	batchPORepo := postgres.NewBatchPORepository(db) // Inisialisasi repository Batch PO
	expenseRepo := postgres.NewExpenseRepository(db) // Inisialisasi repository Expense

	// ==========================================
	// 2. INISIASI USECASE (Layer Logika Bisnis)
	// ==========================================
	userUsecase := usecase.NewUserUsecase(userRepo, storageService, txManager, contextTimeout)
	categoryUsecase := usecase.NewCategoryUsecase(categoryRepo, contextTimeout)
	productUsecase := usecase.NewProductUsecase(productRepo, categoryRepo, storageService, txManager, contextTimeout)
	customerUsecase := usecase.NewCustomerUsecase(customerRepo, userRepo, contextTimeout)
	bankAccountUsecase := usecase.NewBankAccountUsecase(bankAccountRepo, contextTimeout)
	dashboardUsecase := usecase.NewDashboardUsecase(dashboardRepo, contextTimeout)
	reportUsecase := usecase.NewReportUsecase(reportRepo, contextTimeout)
	batchPOUsecase := usecase.NewBatchPOUsecase(batchPORepo, contextTimeout)
	uploadUsecase := usecase.NewUploadUsecase(storageService, contextTimeout)
	expenseUsecase := usecase.NewExpenseUsecase(expenseRepo, contextTimeout) // Inisialisasi usecase Expense

	// Order butuh banyak dependensi untuk validasi bisnis
	orderUsecase := usecase.NewOrderUsecase(orderRepo, customerRepo, userRepo, productRepo, batchPORepo, paymentRepo, txManager, contextTimeout)

	// Payment butuh Order & BankAccount untuk kalkulasi status lunas
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo, orderRepo, bankAccountRepo, txManager, batchPORepo, contextTimeout)

	specTemplateUsecase := usecase.NewSpecTemplateUsecase(specTemplateRepo, contextTimeout)
	// ==========================================
	// 3. INISIASI HANDLER (Layer Delivery)
	// ==========================================
	userHandler := myHttp.NewUserHandler(userUsecase)
	categoryHandler := myHttp.NewCategoryHandler(categoryUsecase)
	productHandler := myHttp.NewProductHandler(productUsecase)
	customerHandler := myHttp.NewCustomerHandler(customerUsecase)
	bankAccountHandler := myHttp.NewBankAccountHandler(bankAccountUsecase)
	orderHandler := myHttp.NewOrderHandler(orderUsecase)
	paymentHandler := myHttp.NewPaymentHandler(paymentUsecase)
	specTemplateHandler := myHttp.NewSpecTemplateHandler(specTemplateUsecase)
	dashboardHandler := myHttp.NewDashboardHandler(dashboardUsecase) // Inisialisasi handler dashboard
	reportHandler := myHttp.NewReportHandler(reportUsecase)
	batchPOHandler := myHttp.NewBatchPOHandler(batchPOUsecase) // Inisialisasi handler Batch PO
	uploadHandler := myHttp.NewUploadHandler(uploadUsecase)
	expenseHandler := myHttp.NewExpenseHandler(expenseUsecase) // Inisialisasi handler Expense

	// ==========================================
	// 4. BUNGKUS KE DALAM STRUCT REGISTRY ROUTER
	// ==========================================
	handlers := myHttp.Handlers{
		UserHandler:         userHandler,
		CategoryHandler:     categoryHandler,
		CustomerHandler:     customerHandler,
		ProductHandler:      productHandler,
		BankAccountHandler:  bankAccountHandler,
		OrderHandler:        orderHandler,
		PaymentHandler:      paymentHandler,
		SpecTemplateHandler: specTemplateHandler,
		DashboardHandler:    dashboardHandler, // Tambahkan handler dashboard ke registry
		ReportHandler:       reportHandler,
		BatchPOHandler:      batchPOHandler, // Tambahkan handler Batch PO ke registry
		UploadHandler:       uploadHandler,
		ExpenseHandler:      expenseHandler, // Tambahkan handler Expense ke registry
	}

	// ==========================================
	// 5. SETUP FIBER APP
	// ==========================================
	app := fiber.New(fiber.Config{
		// Konfigurasi fiber tambahan jika diperlukan
		DisableStartupMessage: true,
	})

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "*"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     frontendURL,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: false,
	}))

	// app.Use(logger.New(logger.Config{
	// 	Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
	// 	TimeFormat: "2006-01-02 15:04:05",
	// 	TimeZone:   "Asia/Jakarta",
	// }))

	app.Use(requestid.New())

	app.Use(middleware.SlogMiddleware())

	// Setup Swagger
	swaggerHost := os.Getenv("SWAGGER_HOST")
	if swaggerHost != "" {
		docs.SwaggerInfo.Host = swaggerHost
	}
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Daftarkan semua route dari router.go
	myHttp.SetupRoutes(app, handlers)

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

	// Menangkap sinyal OS untuk mematikan server dengan aman (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit // Menunggu sinyal masuk

	slog.Info("Menerima sinyal mati, mematikan server dengan sopan...")

	// Memberi waktu maksimal 10 detik untuk menyelesaikan request yang sedang berjalan
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		slog.Error("Server dipaksa mati karena timeout atau error", slog.String("detail", err.Error()))
	}

	slog.Info("✅ SIKOn API Server berhasil dimatikan dengan aman. Sampai jumpa!")
}
