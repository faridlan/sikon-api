package tests

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/faridlan/sikon-api/internal/repository/postgres"
)

// SeedCategory membuat satu data kategori langsung ke Database dan mengembalikan modelnya
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

// SeedOrder mencetak 1 Order beserta 1 OrderItem di dalamnya
func SeedOrder(db *gorm.DB, customerID, salesID, productID string) postgres.OrderModel {
	orderID := uuid.New().String()

	order := postgres.OrderModel{
		ID: orderID,
		// UBAH BARIS INI AGAR SELALU UNIK
		OrderNumber:   "ORD-" + uuid.New().String()[:8],
		CustomerID:    customerID,
		SalesID:       salesID,
		TotalAmount:   100000,
		ShippingCost:  10000,
		OrderStatus:   "pending",
		PaymentStatus: "unpaid",
	}
	db.Create(&order)

	orderItem := postgres.OrderItemModel{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		ProductID: productID,
		Qty:       2,
		Price:     50000,
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
		ReferenceNumber: "TRX-" + uuid.New().String()[:8], // Generate random string kecil
		PaymentType:     paymentType,
	}
	db.Create(&payment)
	return payment
}
