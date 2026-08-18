package http

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/middleware"
	"github.com/faridlan/sikon-api/internal/domain"
	// swagger "github.com/gofiber/swagger"
)

// Handlers struct menampung semua instance handler untuk mempermudah Dependency Injection
type Handlers struct {
	AuthHandler         AuthHandler
	UserHandler         UserHandler
	CategoryHandler     CategoryHandler
	CustomerHandler     CustomerHandler
	ProductHandler      ProductHandler
	BankAccountHandler  BankAccountHandler
	OrderHandler        OrderHandler
	PaymentHandler      PaymentHandler
	SpecTemplateHandler SpecTemplateHandler
	DashboardHandler    DashboardHandler
	ReportHandler       ReportHandler
	BatchPOHandler      BatchPOHandler
	UploadHandler       UploadHandler
	ExpenseHandler      ExpenseHandler
	SeederHandler       SeederHandler
}

// SetupRoutes mengatur seluruh rute API aplikasi SIKOn menggunakan parameter struct Handlers & jwtSecret
func SetupRoutes(app *fiber.App, handlers Handlers, jwtSecret string) {
	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to SIKOn API (Sistem Integrasi Konveksi Online)",
			"version": "2.0",
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "OK",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Swagger Route (Uncomment setelah menjalankan 'swag init' dan menambahkan import)
	// app.Get("/swagger/*", swagger.HandlerDefault)

	// API V1 Group
	api := app.Group("/api")

	// Middleware Instances
	authMiddleware := middleware.JWTMiddleware(jwtSecret)

	// Role Helpers
	guardOwner := middleware.RoleGuard(domain.RoleOwner)
	guardFinance := middleware.RoleGuard(domain.RoleAccounting)
	guardSales := middleware.RoleGuard(domain.RoleSales)
	guardInternal := middleware.RoleGuard(domain.RoleAccounting, domain.RoleSales)

	// --- Auth Routes (Public) ---
	auth := api.Group("/auth")
	auth.Post("/login", handlers.AuthHandler.Login)

	// --- Public Catalog & Sales Directory Routes ---
	usersPublic := api.Group("/users")
	usersPublic.Get("/public/sales", handlers.UserHandler.GetPublicSalesList)

	categoriesPublic := api.Group("/categories")
	categoriesPublic.Get("/", handlers.CategoryHandler.ListCategories)
	categoriesPublic.Get("/:id", handlers.CategoryHandler.GetCategory)

	productsPublic := api.Group("/products")
	productsPublic.Get("/", handlers.ProductHandler.ListProducts)
	productsPublic.Get("/slug/:slug", handlers.ProductHandler.GetProductBySlug)
	productsPublic.Get("/:id", handlers.ProductHandler.GetProduct)

	seederGroup := api.Group("/seeder")
	seederGroup.Post("/generate", handlers.SeederHandler.Generate)
	seederGroup.Post("/clear", handlers.SeederHandler.Clear)

	// =========================================================================
	// PROTECTED ROUTES (Membutuhkan JWT Token)
	// =========================================================================
	protected := api.Use(authMiddleware)

	// --- Users Routes (Owner Full Access) ---
	users := protected.Group("/users")
	users.Post("/register", guardOwner, handlers.UserHandler.Register)
	users.Get("/", guardOwner, handlers.UserHandler.ListUsers)
	users.Get("/:id", guardInternal, handlers.UserHandler.GetProfile)
	users.Put("/:id", guardOwner, handlers.UserHandler.UpdateUser)
	users.Delete("/:id", guardOwner, handlers.UserHandler.DeleteUser)

	// --- Categories Routes (Write oleh Owner) ---
	categories := protected.Group("/categories")
	categories.Post("/", guardOwner, handlers.CategoryHandler.CreateCategory)
	categories.Put("/:id", guardOwner, handlers.CategoryHandler.UpdateCategory)
	categories.Delete("/:id", guardOwner, handlers.CategoryHandler.DeleteCategory)

	// --- Products Routes (Write oleh Owner) ---
	products := protected.Group("/products")
	products.Post("/", guardOwner, handlers.ProductHandler.CreateProduct)
	products.Put("/:id", guardOwner, handlers.ProductHandler.UpdateProduct)
	products.Delete("/:id", guardOwner, handlers.ProductHandler.DeleteProduct)

	// --- Customers Routes (Internal: Accounting & Sales) ---
	customers := protected.Group("/customers")
	customers.Post("/", guardInternal, handlers.CustomerHandler.CreateCustomer)
	customers.Get("/", guardInternal, handlers.CustomerHandler.ListCustomers)
	customers.Get("/:id", guardInternal, handlers.CustomerHandler.GetCustomer)
	customers.Put("/:id", guardInternal, handlers.CustomerHandler.UpdateCustomer)
	customers.Delete("/:id", guardOwner, handlers.CustomerHandler.DeleteCustomer)

	// --- Bank Accounts Routes ---
	bankAccounts := protected.Group("/bank-accounts")
	bankAccounts.Get("/global", handlers.BankAccountHandler.GetGlobalAccounts) // Untuk dropdown transaksi
	bankAccounts.Get("/", guardFinance, handlers.BankAccountHandler.ListAccounts)
	bankAccounts.Get("/:id", guardFinance, handlers.BankAccountHandler.GetByID)
	bankAccounts.Get("/user/:user_id", guardFinance, handlers.BankAccountHandler.GetUserAccounts)
	bankAccounts.Post("/", guardOwner, handlers.BankAccountHandler.CreateAccount)
	bankAccounts.Put("/:id", guardOwner, handlers.BankAccountHandler.UpdateAccount)
	bankAccounts.Delete("/:id", guardOwner, handlers.BankAccountHandler.DeleteAccount)

	// --- Orders Routes (Internal: Sales Create/Read, Accounting Read/Status) ---
	orders := protected.Group("/orders")
	orders.Post("/", guardInternal, handlers.OrderHandler.CreateOrder)
	orders.Get("/", guardInternal, handlers.OrderHandler.ListOrders)
	orders.Get("/:id", guardInternal, handlers.OrderHandler.GetOrder)
	orders.Put("/:id", guardInternal, handlers.OrderHandler.UpdateOrder)
	orders.Patch("/:id/status", guardInternal, handlers.OrderHandler.UpdateOrderStatus)
	orders.Patch("/:id/payment-status", guardFinance, handlers.OrderHandler.UpdatePaymentStatus)
	orders.Delete("/:id", guardOwner, handlers.OrderHandler.DeleteOrder)

	// Order Items Management
	orders.Post("/:id/items", guardInternal, handlers.OrderHandler.AddOrderItem)
	orders.Put("/:id/items/:itemId", guardInternal, handlers.OrderHandler.UpdateOrderItem)
	orders.Delete("/:id/items/:itemId", guardInternal, handlers.OrderHandler.DeleteOrderItem)

	// --- Payments Routes (Sales Create DP, Accounting Verify) ---
	payments := protected.Group("/payments")
	payments.Post("/", guardInternal, handlers.PaymentHandler.ProcessPayment)
	payments.Get("/", guardInternal, handlers.PaymentHandler.ListPayments)
	payments.Get("/order/:order_id", guardInternal, handlers.PaymentHandler.GetPaymentsByOrderID)
	payments.Get("/:id", guardInternal, handlers.PaymentHandler.GetPayment)
	payments.Put("/:id", guardFinance, handlers.PaymentHandler.UpdatePayment)
	payments.Patch("/:id/verify", guardFinance, handlers.PaymentHandler.VerifyPayment) // Approval Keuangan
	payments.Delete("/:id", guardOwner, handlers.PaymentHandler.DeletePayment)

	// --- Spec Templates Routes (Pengaturan Spesifikasi Produksi) ---
	specTemplates := protected.Group("/spec-templates")
	specTemplates.Get("/", guardInternal, handlers.SpecTemplateHandler.ListSpecTemplates)
	specTemplates.Get("/:id", guardInternal, handlers.SpecTemplateHandler.GetSpecTemplate)
	specTemplates.Post("/", guardOwner, handlers.SpecTemplateHandler.CreateSpecTemplate)
	specTemplates.Put("/:id", guardOwner, handlers.SpecTemplateHandler.UpdateSpecTemplate)
	specTemplates.Delete("/:id", guardOwner, handlers.SpecTemplateHandler.DeleteSpecTemplate)

	// --- Expenses Routes (Khusus Finance/Accounting & Owner) ---
	expenses := protected.Group("/expenses")
	expenses.Get("/categories", guardFinance, handlers.ExpenseHandler.ListCategories)
	expenses.Post("/categories", guardFinance, handlers.ExpenseHandler.CreateCategory)
	expenses.Get("/", guardFinance, handlers.ExpenseHandler.ListExpenses)
	expenses.Post("/", guardFinance, handlers.ExpenseHandler.CreateExpense)
	expenses.Get("/:id", guardFinance, handlers.ExpenseHandler.GetExpense)
	expenses.Delete("/:id", guardOwner, handlers.ExpenseHandler.DeleteExpense)

	// --- Batch PO Routes ---
	batchPos := protected.Group("/batch-pos")
	batchPos.Get("/active", guardInternal, handlers.BatchPOHandler.ListActiveBatchPOs) // Dropdown transaksi Sales
	batchPos.Get("/", guardInternal, handlers.BatchPOHandler.ListBatchPOs)
	batchPos.Get("/:id", guardInternal, handlers.BatchPOHandler.GetBatchPO)
	batchPos.Post("/", guardOwner, handlers.BatchPOHandler.CreateBatchPO)
	batchPos.Put("/:id", guardOwner, handlers.BatchPOHandler.UpdateBatchPO)
	batchPos.Patch("/:id/status", guardOwner, handlers.BatchPOHandler.UpdateStatus)
	batchPos.Delete("/:id", guardOwner, handlers.BatchPOHandler.DeleteBatchPO)

	// --- Dashboard Routes ---
	dashboard := protected.Group("/dashboard")
	dashboard.Get("/overview", guardInternal, handlers.DashboardHandler.GetOverview)
	dashboard.Get("/summary", guardInternal, handlers.DashboardHandler.GetSummary)
	dashboard.Get("/sales-report", guardSales, handlers.DashboardHandler.GetSalesReport)
	dashboard.Get("/receivables-report", guardFinance, handlers.DashboardHandler.GetReceivablesReport)

	// --- Reports Routes (Khusus Finance/Accounting & Owner) ---
	reports := protected.Group("/reports")
	reports.Get("/accounting", guardFinance, handlers.ReportHandler.GetAccountingReport)
	reports.Get("/production", guardInternal, handlers.ReportHandler.GetProductionReport)
	reports.Get("/po/:po_id/summary", guardInternal, handlers.ReportHandler.GetPOSummaryReport)
	reports.Get("/receivables", guardFinance, handlers.ReportHandler.GetReceivablesReport)
	reports.Get("/daily", guardFinance, handlers.ReportHandler.GetDailyReport)

	// --- Uploads & Seeder Routes ---
	uploads := protected.Group("/uploads")
	uploads.Post("/image", guardInternal, handlers.UploadHandler.UploadImage)

}
