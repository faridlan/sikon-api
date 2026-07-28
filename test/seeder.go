package tests

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/faridlan/sikon-api/internal/repository/postgres"
)

func SeedCategory(db *gorm.DB, name string, image ...string) postgres.CategoryModel {
	if len(image) == 0 {
		image = append(image, "")
	}

	category := postgres.CategoryModel{
		ID:       uuid.New().String(),
		Name:     name,
		ImageURL: image[0],
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

func SeedUser(db *gorm.DB, name, email, role string, image ...string) postgres.UserModel {

	if len(image) == 0 {
		image = append(image, "") // Default kosong jika tidak ada image yang diberikan
	}

	user := postgres.UserModel{
		ID:       uuid.New().String(),
		Name:     name,
		Email:    email,
		Password: "hashedpassword123", // Anggap saja ini sudah di-hash oleh repo/usecase
		Role:     role,
		ImageURL: image[0],
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

// ========================================================================
// 🚨 TAMBAHAN: Seeder untuk Batch PO
// ========================================================================
func SeedBatchPO(db *gorm.DB, name string, status string) postgres.BatchPOModel {
	batchPO := postgres.BatchPOModel{
		ID:        uuid.New().String(),
		Name:      name,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, 0, 7), // Default tutup 7 hari lagi
		Status:    status,
		Quota:     100,
	}
	db.Create(&batchPO)
	return batchPO
}

// Fungsi Helper untuk membuat pointer string dengan mudah
func StringPtr(s string) *string {
	return &s
}

// ========================================================================
// 🚨 UPDATE: Menambahkan parameter batchPoID ke dalam SeedOrder
// ========================================================================
func SeedOrder(db *gorm.DB, batchPoID, customerID, salesID, productID string, customStatus ...string) postgres.OrderModel {
	orderID := uuid.New().String()
	validUntil := time.Now().AddDate(0, 0, 7)

	status := "quotation"
	if len(customStatus) > 0 && customStatus[0] != "" {
		status = customStatus[0]
	}

	// Tangani pointer agar aman jika tidak ada ID yang dikirim
	var batchPoIDPtr *string
	if batchPoID != "" {
		batchPoIDPtr = &batchPoID
	}

	order := postgres.OrderModel{
		ID:              orderID,
		OrderNumber:     "ORD-" + uuid.New().String()[:8],
		BatchPoID:       batchPoIDPtr, // <-- Disematkan di sini
		CustomerID:      customerID,
		SalesID:         salesID,
		Subtotal:        100000,
		TotalAmount:     110000,
		ShippingCost:    10000,
		OrderStatus:     status,
		PaymentStatus:   "unpaid",
		ValidUntil:      &validUntil,
		TermsConditions: "DP Minimal 50%",
	}
	db.Create(&order)

	detailsData := map[string]any{
		"Benang":  "Benang Bordir Menggunakan Benang Polyster",
		"Bordir":  "Bordir Menggunakan Sistem Komputerisasi",
		"Jahitan": "Jahit Rapi",
		"Bahan": map[string]any{
			"Name":  "Katun Baby Canvas",
			"Spec":  "menggunakan baby canvas",
			"Color": "Hitam",
		},
	}

	detailsBytes, _ := json.Marshal(detailsData)

	orderItem := postgres.OrderItemModel{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		ProductID: productID,
		Qty:       2,
		Price:     50000,
		Details:   datatypes.JSON(detailsBytes),
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

func SeedSpecTemplate(db *gorm.DB, name, spec string) postgres.SpecTemplateModel {
	specTemplate := postgres.SpecTemplateModel{
		ID:   uuid.New().String(),
		Name: name,
		Spec: spec,
	}
	db.Create(&specTemplate)
	return specTemplate
}

func SeedExpenseCategory(db *gorm.DB, name, expenseType, description string) postgres.ExpenseCategoryModel {
	category := postgres.ExpenseCategoryModel{
		ID:          uuid.New().String(),
		Name:        name,
		Type:        expenseType,
		Description: description,
	}
	db.Create(&category)
	return category
}

func SeedExpense(db *gorm.DB, categoryID, createdByID string, batchPoID *string, title string, amount float64, expenseDate time.Time) postgres.ExpenseModel {
	expense := postgres.ExpenseModel{
		ID:                uuid.New().String(),
		ExpenseCategoryID: categoryID,
		BatchPoID:         batchPoID,
		Title:             title,
		Amount:            amount,
		ExpenseDate:       expenseDate,
		Notes:             "catatan test",
		CreatedByID:       createdByID,
	}
	db.Create(&expense)
	return expense
}
