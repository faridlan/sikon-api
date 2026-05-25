package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
	// Nanti Anda akan mengimpor module Swagger fiber di sini setelah setup selesai
	// swagger "github.com/gofiber/swagger"
)

// Handlers struct menampung semua instance handler untuk mempermudah Dependency Injection
type Handlers struct {
	UserHandler        UserHandler
	CategoryHandler    CategoryHandler
	CustomerHandler    CustomerHandler
	ProductHandler     ProductHandler
	BankAccountHandler BankAccountHandler
	OrderHandler       OrderHandler
	PaymentHandler     PaymentHandler
}

// SetupRoutes mengatur seluruh rute API aplikasi SIKOn menggunakan parameter struct Handlers
func SetupRoutes(app *fiber.App, handlers Handlers) {
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

	// --- Users Routes ---
	users := api.Group("/users")
	users.Post("/register", handlers.UserHandler.Register)
	users.Get("/", handlers.UserHandler.ListUsers)
	users.Get("/:id", handlers.UserHandler.GetProfile)
	users.Put("/:id", handlers.UserHandler.UpdateUser)
	users.Delete("/:id", handlers.UserHandler.DeleteUser)

	// --- Categories Routes ---
	categories := api.Group("/categories")
	categories.Post("/", handlers.CategoryHandler.CreateCategory)
	categories.Get("/", handlers.CategoryHandler.ListCategories)
	categories.Get("/:id", handlers.CategoryHandler.GetCategory)
	categories.Put("/:id", handlers.CategoryHandler.UpdateCategory)
	categories.Delete("/:id", handlers.CategoryHandler.DeleteCategory)

	// --- Customers Routes ---
	customers := api.Group("/customers")
	customers.Post("/", handlers.CustomerHandler.CreateCustomer)
	customers.Get("/", handlers.CustomerHandler.ListCustomers)
	customers.Get("/:id", handlers.CustomerHandler.GetCustomer)
	customers.Put("/:id", handlers.CustomerHandler.UpdateCustomer)
	customers.Delete("/:id", handlers.CustomerHandler.DeleteCustomer)

	// --- Products Routes ---
	products := api.Group("/products")
	products.Post("/", handlers.ProductHandler.CreateProduct)
	products.Get("/", handlers.ProductHandler.ListProducts)
	products.Get("/:id", handlers.ProductHandler.GetProduct)
	products.Put("/:id", handlers.ProductHandler.UpdateProduct)
	products.Delete("/:id", handlers.ProductHandler.DeleteProduct)

	// --- Bank Accounts Routes ---
	bankAccounts := api.Group("/bank-accounts")
	bankAccounts.Post("/", handlers.BankAccountHandler.CreateAccount)
	bankAccounts.Put("/:id", handlers.BankAccountHandler.UpdateAccount)
	bankAccounts.Get("/", handlers.BankAccountHandler.ListAccounts)
	bankAccounts.Get("/global", handlers.BankAccountHandler.GetGlobalAccounts)
	bankAccounts.Get("/:id", handlers.BankAccountHandler.GetByID)
	bankAccounts.Get("/user/:user_id", handlers.BankAccountHandler.GetUserAccounts)
	bankAccounts.Delete("/:id", handlers.BankAccountHandler.DeleteAccount)

	// --- Orders Routes ---
	orders := api.Group("/orders")
	orders.Post("/", handlers.OrderHandler.CreateOrder)
	orders.Get("/", handlers.OrderHandler.ListOrders)
	orders.Get("/:id", handlers.OrderHandler.GetOrder)
	orders.Put("/:id", handlers.OrderHandler.UpdateOrder)
	orders.Patch("/:id/status", handlers.OrderHandler.UpdateOrderStatus)
	orders.Delete("/:id", handlers.OrderHandler.DeleteOrder)

	// --- Payments Routes ---
	payments := api.Group("/payments")
	payments.Post("/", handlers.PaymentHandler.ProcessPayment)
	payments.Get("/", handlers.PaymentHandler.ListPayments)
	payments.Get("/order/:order_id", handlers.PaymentHandler.GetPaymentsByOrderID)
	payments.Get("/:id", handlers.PaymentHandler.GetPayment)
	payments.Put("/:id", handlers.PaymentHandler.UpdatePayment)
	payments.Delete("/:id", handlers.PaymentHandler.DeletePayment)
}
