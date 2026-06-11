package tests

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/faridlan/sikon-api/internal/repository/postgres"
)

// ... (Fungsi SeedCategory, SeedProduct, SeedUser tetap ada) ...
func SeedCategory(db *gorm.DB, name string) postgres.CategoryModel {
	category := postgres.CategoryModel{
		ID:   uuid.New().String(),
		Name: name,
	}
	db.Create(&category)
	return category
}

func SeedProduct(db *gorm.DB, categoryID string, name string, price float64) postgres.ProductModel {
	product := postgres.ProductModel{
		ID:          uuid.New().String(),
		CategoryID:  categoryID,
		Name:        name,
		Description: "Deskripsi " + name,
		BasePrice:   price,
	}
	db.Create(&product)
	return product
}

func SeedUser(db *gorm.DB, name, email, role string) postgres.UserModel {
	user := postgres.UserModel{
		ID:       uuid.New().String(),
		Name:     name,
		Email:    email,
		Password: "hashedpassword123", // Anggap saja ini sudah di-hash oleh repo/usecase
		Role:     role,
	}
	db.Create(&user)
	return user
}

func SeedCustomer(db *gorm.DB, name, phone, address string) postgres.CustomerModel {
	customer := postgres.CustomerModel{
		ID:      uuid.New().String(),
		Name:    name,
		Phone:   phone,
		Address: address,
	}

	// Gunakan Omit agar GORM tidak memaksa mengirim string kosong ("")
	// ke kolom created_by yang bertipe UUID
	db.Omit("created_by").Create(&customer)

	return customer
}

// TAMBAHAN: Fungsi seeder baru untuk customer yang memiliki SalesID
func SeedCustomerWithSales(db *gorm.DB, name, phone, address string, salesID string) postgres.CustomerModel {
	customer := postgres.CustomerModel{
		ID:      uuid.New().String(),
		Name:    name,
		Phone:   phone,
		Address: address,
		SalesID: &salesID, // Memasukkan pointer string
	}

	db.Omit("created_by").Create(&customer)

	return customer
}

func SeedBankAccount(db *gorm.DB, userID *string, bankName, accNumber, accName string) postgres.BankAccountModel {
	account := postgres.BankAccountModel{
		ID:            uuid.New().String(),
		UserID:        userID,
		BankName:      bankName,
		AccountNumber: accNumber,
		AccountName:   accName,
	}
	db.Create(&account)
	return account
}

// Fungsi Helper untuk membuat pointer string dengan mudah
func StringPtr(s string) *string {
	return &s
}

func SeedOrder(db *gorm.DB, customerID, salesID, productID string, customStatus ...string) postgres.OrderModel {
	orderID := uuid.New().String()
	validUntil := time.Now().AddDate(0, 0, 7)

	status := "quotation"
	if len(customStatus) > 0 && customStatus[0] != "" {
		status = customStatus[0]
	}

	order := postgres.OrderModel{
		ID:           orderID,
		OrderNumber:  "ORD-" + uuid.New().String()[:8],
		CustomerID:   customerID,
		SalesID:      salesID,
		Subtotal:     100000, // <-- TAMBAHAN: Harga 2 item @ 50.000
		TotalAmount:  110000, // <-- UPDATE: Subtotal (100.000) + ShippingCost (10.000)
		ShippingCost: 10000,
		// Tax & Discount biarkan kosong (default 0)
		OrderStatus:     status,
		PaymentStatus:   "unpaid",
		ValidUntil:      &validUntil,
		TermsConditions: "DP Minimal 50%",
	}
	db.Create(&order)

	orderItem := postgres.OrderItemModel{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		ProductID: productID,
		Qty:       2,
		Price:     50000,
		Details:   map[string]any{"Benang": "Benang Bordir Menggunakan Benang Polyster", "Bordir": "Bordir Menggunakan Sistem Komputerisasi", "Jahitan": "Jahit Rapi", "Bahan": map[string]any{"Name": "Katun Baby Canvas", "Spec": "menggunakan baby canvas", "Color": "Hitam"}},
	}
	db.Create(&orderItem)

	return order
}
func SeedPayment(db *gorm.DB, orderID, bankAccountID string, amount float64, paymentType string) postgres.PaymentModel {
	payment := postgres.PaymentModel{
		ID:              uuid.New().String(),
		OrderID:         orderID,
		BankAccountID:   bankAccountID,
		Amount:          amount,
		PaymentDate:     time.Now(),
		ReferenceNumber: "TRX-" + uuid.New().String()[:8],
		PaymentType:     paymentType,
	}
	db.Create(&payment)
	return payment
}
