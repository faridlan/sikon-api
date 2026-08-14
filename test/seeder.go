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

// SeedProduct sederhana untuk kemeja/kaos dasar
func SeedProduct(db *gorm.DB, categoryID string, name string, price float64) postgres.ProductModel {
	product := postgres.ProductModel{
		ID:            uuid.New().String(),
		CategoryID:    categoryID,
		Name:          name,
		Description:   "Deskripsi " + name,
		BasePrice:     price,
		Slug:          "slug-" + uuid.New().String()[:8],
		GSMInfo:       "210gsm",
		FabricSummary: "Ripstop Cotton",
		Rating:        4.9,
		SoldCount:     100,
		ReviewCount:   25,
		KeyFeatures:   `["Bahan anti robek", "Dual chest pocket"]`,
	}
	db.Create(&product)
	return product
}

// ========================================================================
// 🚨 SEEDER TAMBAHAN: Seed Custom Product dengan Fabric, Color, Wholesale, & Designer
// ========================================================================
func SeedFullCustomProduct(db *gorm.DB, categoryID string, name string, price float64) postgres.ProductModel {
	productID := uuid.New().String()
	fabricID := uuid.New().String()
	designerID := uuid.New().String()

	product := postgres.ProductModel{
		ID:            productID,
		CategoryID:    categoryID,
		Name:          name,
		Description:   "Deskripsi lengkap " + name,
		BasePrice:     price,
		Slug:          "custom-slug-" + uuid.New().String()[:8],
		GSMInfo:       "210gsm",
		FabricSummary: "Ripstop Cotton 65/35",
		Rating:        4.9,
		SoldCount:     50,
		ReviewCount:   10,
		KeyFeatures:   `["Bahan anti robek", "Dual chest pocket velcro", "Ventilasi punggung"]`,
		Images: []postgres.ProductImageModel{
			{
				ID:        uuid.New().String(),
				ProductID: productID,
				ImageURL:  "https://example.com/tactical-primary.jpg",
				IsPrimary: true,
			},
		},
		Fabrics: []postgres.ProductFabricModel{
			{
				ID:              fabricID,
				ProductID:       productID,
				Name:            "Ripstop Cotton 65/35",
				Description:     "Kuat & anti robek, 210gsm",
				Composition:     "65% Cotton / 35% Polyester",
				CareInstruction: "Cuci mesin air dingin",
				BasePrice:       price,
				PriceAdjustment: 0,
				IsDefault:       true,
				Colors: []postgres.FabricColorModel{
					{
						ID:       uuid.New().String(),
						FabricID: &fabricID,
						Name:     "Olive",
						HexCode:  "#4b5320",
					},
					{
						ID:       uuid.New().String(),
						FabricID: &fabricID,
						Name:     "Navy",
						HexCode:  "#1b263b",
					},
				},
			},
		},
		Wholesale: []postgres.WholesalePriceModel{
			{
				ID:        uuid.New().String(),
				ProductID: productID,
				FabricID:  &fabricID,
				MinQty:    6,
				UnitPrice: price - 10000,
			},
		},
		DesignModel: &postgres.DesignerModel{
			ID:          designerID,
			ProductID:   productID,
			Name:        "Series 1 — Lengan Panjang",
			Type:        "long_sleeve",
			Description: "Template kemeja taktikal lengan panjang",
			Views: []postgres.ProductModelViewModel{
				{
					ID:             uuid.New().String(),
					ProductModelID: designerID,
					Side:           "front",
					ArtURL:         "https://example.com/mockups/front-art.png",
					MaskURL:        "https://example.com/mockups/front-mask.png",
					Width:          1756,
					Height:         1920,
				},
				{
					ID:             uuid.New().String(),
					ProductModelID: designerID,
					Side:           "back",
					ArtURL:         "https://example.com/mockups/back-art.png",
					MaskURL:        "https://example.com/mockups/back-mask.png",
					Width:          1738,
					Height:         1920,
				},
			},
		},
	}

	db.Create(&product)
	return product
}

func SeedUser(db *gorm.DB, name, email, role string, image ...string) postgres.UserModel {
	if len(image) == 0 {
		image = append(image, "")
	}

	user := postgres.UserModel{
		ID:       uuid.New().String(),
		Name:     name,
		Email:    email,
		Password: "hashedpassword123",
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

	db.Omit("created_by").Create(&customer)
	return customer
}

func SeedCustomerWithSales(db *gorm.DB, name, phone, address string, salesID string) postgres.CustomerModel {
	customer := postgres.CustomerModel{
		ID:      uuid.New().String(),
		Name:    name,
		Phone:   phone,
		Address: address,
		SalesID: &salesID,
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

func SeedBatchPO(db *gorm.DB, name string, status string) postgres.BatchPOModel {
	batchPO := postgres.BatchPOModel{
		ID:        uuid.New().String(),
		Name:      name,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(0, 0, 7),
		Status:    status,
		Quota:     100,
	}
	db.Create(&batchPO)
	return batchPO
}

func StringPtr(s string) *string {
	return &s
}

func SeedOrder(db *gorm.DB, batchPoID, customerID, salesID, productID string, customStatus ...string) postgres.OrderModel {
	orderID := uuid.New().String()
	validUntil := time.Now().AddDate(0, 0, 7)

	status := "quotation"
	if len(customStatus) > 0 && customStatus[0] != "" {
		status = customStatus[0]
	}

	var batchPoIDPtr *string
	if batchPoID != "" {
		batchPoIDPtr = &batchPoID
	}

	order := postgres.OrderModel{
		ID:              orderID,
		OrderNumber:     "ORD-" + uuid.New().String()[:8],
		BatchPoID:       batchPoIDPtr,
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
