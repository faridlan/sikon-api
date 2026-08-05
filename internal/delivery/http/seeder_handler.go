package http

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/faridlan/sikon-api/internal/domain"
	postgresRepo "github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/utils"
)

type SeederHandler interface {
	Generate(c *fiber.Ctx) error
	Clear(c *fiber.Ctx) error
}

type seederHandler struct {
	db *gorm.DB
}

func NewSeederHandler(db *gorm.DB) SeederHandler {
	return &seederHandler{db: db}
}

// @Summary Hapus Semua Data Seeder
// @Tags Seeder
// @Router /seeder/clear [post]
func (h *seederHandler) Clear(c *fiber.Ctx) error {
	// Hapus Relasi Produk Baru
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.ProductModelViewModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.DesignerModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.WholesalePriceModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.FabricColorModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.ProductFabricModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.ProductImageModel{})

	// Hapus Data Utama
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.ExpenseModel{})
	h.db.Unscoped().Where("name LIKE ?", "Dummy Cat Exp%").Delete(&postgresRepo.ExpenseCategoryModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.PaymentModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.OrderItemModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.OrderModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.BatchPOModel{})
	h.db.Unscoped().Where("name LIKE ?", "Dummy Cust%").Delete(&postgresRepo.CustomerModel{})
	h.db.Unscoped().Where("name LIKE ?", "Dummy Prod%").Delete(&postgresRepo.ProductModel{})
	h.db.Unscoped().Where("name LIKE ?", "Dummy Cat%").Delete(&postgresRepo.CategoryModel{})
	h.db.Unscoped().Where("email LIKE ?", "%_dummy@sikon.com").Delete(&postgresRepo.UserModel{})
	h.db.Unscoped().Where("account_number = ?", "999888777").Delete(&postgresRepo.BankAccountModel{})

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil membersihkan data seeder", nil)
}

// @Summary Generate Data Testing Realistis
// @Tags Seeder
// @Router /seeder/generate [post]
func (h *seederHandler) Generate(c *fiber.Ctx) error {
	// --- 1. SETUP MASTER DATA ---

	salesDummies := []postgresRepo.UserModel{
		{ID: uuid.NewString(), Name: "Budi (Sales Dummy)", Email: "budi_dummy@sikon.com", Password: "password123", Role: "sales"},
		{ID: uuid.NewString(), Name: "Andi (Sales Dummy)", Email: "andi_dummy@sikon.com", Password: "password123", Role: "sales"},
		{ID: uuid.NewString(), Name: "Cici (Sales Dummy)", Email: "cici_dummy@sikon.com", Password: "password123", Role: "sales"},
	}

	var activeSalesIDs []string
	for _, s := range salesDummies {
		var existing postgresRepo.UserModel
		if err := h.db.Where("email = ?", s.Email).First(&existing).Error; err != nil {
			h.db.Create(&s)
			activeSalesIDs = append(activeSalesIDs, s.ID)
		} else {
			activeSalesIDs = append(activeSalesIDs, existing.ID)
		}
	}

	bankAccount := postgresRepo.BankAccountModel{
		ID:            uuid.NewString(),
		BankName:      "BCA",
		AccountNumber: "999888777",
		AccountName:   "PT Konveksi Dummy",
	}
	h.db.Create(&bankAccount)

	cat := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Dummy Cat Kemeja Taktikal"}
	h.db.Create(&cat)

	// --- SETUP SEEDER PRODUK KATALOG & CANVAS DESIGNER REALISTIS ---

	prod1ID := uuid.NewString()
	prod2ID := uuid.NewString()

	// Fabric IDs
	fab1Prod1 := uuid.NewString()
	fab2Prod1 := uuid.NewString()
	fab1Prod2 := uuid.NewString()

	// Designer Model IDs
	dm1ID := uuid.NewString()
	dm2ID := uuid.NewString()

	product1 := postgresRepo.ProductModel{
		ID:            prod1ID,
		CategoryID:    cat.ID,
		Name:          "Dummy Prod Kemeja Taktikal Premium 7200",
		Description:   "Kemeja taktikal lengan panjang, 2 saku flap velcro, ventilasi punggung",
		BasePrice:     185000,
		Slug:          "dummy-prod-kemeja-taktikal-premium-7200",
		GSMInfo:       "210gsm",
		FabricSummary: "Ripstop Cotton",
		Rating:        4.9,
		SoldCount:     1240,
		ReviewCount:   318,
		KeyFeatures:   `["Bahan ripstop anti robek", "Dual chest pocket velcro", "Pen slot di lengan", "Ventilasi punggung", "Bisa custom bordir & patch"]`,
		Images: []postgresRepo.ProductImageModel{
			{ID: uuid.NewString(), ProductID: prod1ID, ImageURL: "https://raw.githubusercontent.com/faridlan/assets/main/mockups/series1-front-long.png", IsPrimary: true},
			{ID: uuid.NewString(), ProductID: prod1ID, ImageURL: "https://raw.githubusercontent.com/faridlan/assets/main/mockups/series1-back-long.png", IsPrimary: false},
		},
		Fabrics: []postgresRepo.ProductFabricModel{
			{
				ID:              fab1Prod1,
				ProductID:       prod1ID,
				Name:            "Ripstop Cotton 65/35",
				Description:     "Kuat & anti robek, 210gsm",
				Composition:     "65% Cotton / 35% Polyester",
				CareInstruction: "Cuci mesin air dingin, jangan diputihkan, setrika suhu sedang",
				BasePrice:       185000,
				PriceAdjustment: 0,
				IsDefault:       true,
				Colors: []postgresRepo.FabricColorModel{
					{ID: uuid.NewString(), FabricID: fab1Prod1, Name: "Olive", HexCode: "#4b5320"},
					{ID: uuid.NewString(), FabricID: fab1Prod1, Name: "Navy", HexCode: "#1b263b"},
					{ID: uuid.NewString(), FabricID: fab1Prod1, Name: "Black", HexCode: "#000000"},
					{ID: uuid.NewString(), FabricID: fab1Prod1, Name: "Khaki", HexCode: "#c2b280"},
				},
			},
			{
				ID:              fab2Prod1,
				ProductID:       prod1ID,
				Name:            "Nagata Drill Premium",
				Description:     "Tebal & kokoh, 260gsm",
				Composition:     "100% Cotton",
				CareInstruction: "Cuci mesin air dingin, setrika suhu sedang",
				BasePrice:       200000,
				PriceAdjustment: 15000,
				IsDefault:       false,
				Colors: []postgresRepo.FabricColorModel{
					{ID: uuid.NewString(), FabricID: fab2Prod1, Name: "Black", HexCode: "#000000"},
					{ID: uuid.NewString(), FabricID: fab2Prod1, Name: "Navy", HexCode: "#1b263b"},
				},
			},
		},
		Wholesale: []postgresRepo.WholesalePriceModel{
			{ID: uuid.NewString(), ProductID: prod1ID, FabricID: &fab1Prod1, MinQty: 6, MaxQty: pointerInt(23), UnitPrice: 175000},
			{ID: uuid.NewString(), ProductID: prod1ID, FabricID: &fab1Prod1, MinQty: 24, MaxQty: nil, UnitPrice: 165000},
		},
		DesignModel: &postgresRepo.DesignerModel{
			ID:          dm1ID,
			ProductID:   prod1ID,
			Name:        "Series 1 — Lengan Panjang",
			Type:        "long_sleeve",
			Description: "Template kemeja taktikal 2 saku flap",
			Views: []postgresRepo.ProductModelViewModel{
				{
					ID:             uuid.NewString(),
					ProductModelID: dm1ID,
					Side:           "front",
					ArtURL:         "/mockups/series1-front-long.png",
					MaskURL:        "/mockups/series1-front-mask-long.png",
					Width:          1756,
					Height:         1920,
				},
				{
					ID:             uuid.NewString(),
					ProductModelID: dm1ID,
					Side:           "back",
					ArtURL:         "/mockups/series1-back-long.png",
					MaskURL:        "/mockups/series1-back-mask-long.png",
					Width:          1738,
					Height:         1920,
				},
			},
		},
	}

	product2 := postgresRepo.ProductModel{
		ID:            prod2ID,
		CategoryID:    cat.ID,
		Name:          "Dummy Prod Kemeja PDL Lengan Pendek 5000",
		Description:   "Kemeja taktikal lengan pendek, ringan & dingin untuk aktivitas lapangan",
		BasePrice:     165000,
		Slug:          "dummy-prod-kemeja-pdl-lengan-pendek-5000",
		GSMInfo:       "190gsm",
		FabricSummary: "American Drill",
		Rating:        4.8,
		SoldCount:     760,
		ReviewCount:   180,
		KeyFeatures:   `["Bahan halus & dingin", "Jahitan double stitch", "Tahan cuci berulang"]`,
		Images: []postgresRepo.ProductImageModel{
			{ID: uuid.NewString(), ProductID: prod2ID, ImageURL: "https://raw.githubusercontent.com/faridlan/assets/main/mockups/series1-front-short.png", IsPrimary: true},
		},
		Fabrics: []postgresRepo.ProductFabricModel{
			{
				ID:              fab1Prod2,
				ProductID:       prod2ID,
				Name:            "American Drill",
				Description:     "Halus & nyaman, 230gsm",
				Composition:     "80% Cotton / 20% Polyester",
				CareInstruction: "Cuci mesin air dingin",
				BasePrice:       165000,
				PriceAdjustment: 0,
				IsDefault:       true,
				Colors: []postgresRepo.FabricColorModel{
					{ID: uuid.NewString(), FabricID: fab1Prod2, Name: "Olive", HexCode: "#4b5320"},
					{ID: uuid.NewString(), FabricID: fab1Prod2, Name: "Black", HexCode: "#000000"},
				},
			},
		},
		Wholesale: []postgresRepo.WholesalePriceModel{
			{ID: uuid.NewString(), ProductID: prod2ID, FabricID: nil, MinQty: 12, MaxQty: nil, UnitPrice: 155000},
		},
		DesignModel: &postgresRepo.DesignerModel{
			ID:          dm2ID,
			ProductID:   prod2ID,
			Name:        "Series 1 — Lengan Pendek",
			Type:        "short_sleeve",
			Description: "Template kemeja taktikal lengan pendek",
			Views: []postgresRepo.ProductModelViewModel{
				{
					ID:             uuid.NewString(),
					ProductModelID: dm2ID,
					Side:           "front",
					ArtURL:         "/mockups/series1-front-short.png",
					MaskURL:        "/mockups/series1-front-mask-short.png",
					Width:          1727,
					Height:         1920,
				},
				{
					ID:             uuid.NewString(),
					ProductModelID: dm2ID,
					Side:           "back",
					ArtURL:         "/mockups/series1-back-short.png",
					MaskURL:        "/mockups/series1-back-mask-short.png",
					Width:          1708,
					Height:         1920,
				},
			},
		},
	}

	h.db.Create(&product1)
	h.db.Create(&product2)
	products := []postgresRepo.ProductModel{product1, product2}

	// D. Buat Customers dan distribusikan secara ADIL (Round-Robin)
	var customers []postgresRepo.CustomerModel
	customerNames := []string{"Dummy Cust PT A", "Dummy Cust PT B", "Dummy Cust Personal C", "Dummy Cust CV D", "Dummy Cust Personal E", "Dummy Cust CV F"}

	for i, name := range customerNames {
		assignedSalesID := activeSalesIDs[i%len(activeSalesIDs)]
		cust := postgresRepo.CustomerModel{
			ID:        uuid.NewString(),
			Name:      name,
			Phone:     fmt.Sprintf("08111222%d", i),
			CreatedBy: assignedSalesID,
			SalesID:   &assignedSalesID,
		}
		h.db.Create(&cust)
		customers = append(customers, cust)
	}

	// E. Buat Kategori Pengeluaran
	expenseCats := []postgresRepo.ExpenseCategoryModel{
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Belanja Kain & Benang", Type: "hpp"},
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Ongkos Jahit (CMT)", Type: "hpp"},
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Operasional Listrik", Type: "operational"},
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Biaya Iklan Meta Ads", Type: "operational"},
	}
	for _, ec := range expenseCats {
		h.db.Create(&ec)
	}

	// --- 2. SETUP BATCH PO ---
	poSchedules := []struct {
		Name   string
		Month  int
		Year   int
		Start  string
		End    string
		Status string
	}{
		{"PO 1 JULI 2026", 7, 2026, "2026-06-27", "2026-07-04", string(domain.BatchPOStatusClosed)},
		{"PO 2 JULI 2026", 7, 2026, "2026-07-04", "2026-07-11", string(domain.BatchPOStatusClosed)},
		{"PO 3 JULI 2026", 7, 2026, "2026-07-11", "2026-07-18", string(domain.BatchPOStatusClosed)},
		{"PO 4 JULI 2026", 7, 2026, "2026-07-18", "2026-07-25", string(domain.BatchPOStatusClosed)},
		{"PO 1 AGUSTUS 2026", 8, 2026, "2026-07-25", "2026-08-01", string(domain.BatchPOStatusClosed)},
		{"PO 2 AGUSTUS 2026", 8, 2026, "2026-08-01", "2026-08-08", string(domain.BatchPOStatusActive)},
	}

	var batchPOs []postgresRepo.BatchPOModel
	for _, p := range poSchedules {
		start, _ := time.Parse("2006-01-02", p.Start)
		end, _ := time.Parse("2006-01-02", p.End)

		po := postgresRepo.BatchPOModel{
			ID:          uuid.NewString(),
			Name:        p.Name,
			Status:      p.Status,
			Quota:       500,
			TargetMonth: p.Month,
			TargetYear:  p.Year,
			StartDate:   start,
			EndDate:     end,
		}
		h.db.Create(&po)
		batchPOs = append(batchPOs, po)
	}

	// --- 3. LOOPING TRANSAKSI HARIAN ---
	startDate, _ := time.Parse("2006-01-02", "2026-06-27")
	endDate, _ := time.Parse("2006-01-02", "2026-08-03")

	orderCounter := 1
	expenseCounter := 0

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		var activePoID *string
		for _, po := range batchPOs {
			if (d.Equal(po.StartDate) || d.After(po.StartDate)) && (d.Before(po.EndDate) || d.Equal(po.EndDate)) {
				poID := po.ID
				activePoID = &poID
				break
			}
		}

		if activePoID == nil {
			continue
		}

		numOrders := rand.Intn(3) + 1
		for i := 0; i < numOrders; i++ {
			cust := customers[rand.Intn(len(customers))]
			prod := products[rand.Intn(len(products))]

			qty := rand.Intn(10) + 1
			totalAmount := float64(qty) * prod.BasePrice

			payTypeRand := rand.Intn(3)
			var paymentStatus string
			var paidAmount float64

			switch payTypeRand {
			case 0:
				paymentStatus = string(domain.PaymentStatusPaid)
				paidAmount = totalAmount
			case 1:
				paymentStatus = string(domain.PaymentStatusPartial)
				paidAmount = totalAmount / 2
			case 2:
				paymentStatus = string(domain.PaymentStatusUnpaid)
				paidAmount = 0
			}

			approvedTime := d.Add(10 * time.Hour)
			orderID := uuid.NewString()

			order := postgresRepo.OrderModel{
				ID:            orderID,
				OrderNumber:   fmt.Sprintf("SEED-ORD-%04d", orderCounter),
				BatchPoID:     activePoID,
				CustomerID:    cust.ID,
				SalesID:       *cust.SalesID,
				Subtotal:      totalAmount,
				TotalAmount:   totalAmount,
				OrderStatus:   string(domain.OrderStatusProduction),
				PaymentStatus: paymentStatus,
				CreatedAt:     approvedTime,
				ApprovedAt:    &approvedTime,
			}
			h.db.Create(&order)

			orderItem := postgresRepo.OrderItemModel{
				ID:        uuid.NewString(),
				OrderID:   orderID,
				ProductID: prod.ID,
				Qty:       qty,
				Price:     prod.BasePrice,
			}
			h.db.Create(&orderItem)

			if paidAmount > 0 {
				payTime := approvedTime.Add(2 * time.Hour)
				verifierID := *cust.SalesID

				payment := postgresRepo.PaymentModel{
					ID:              uuid.NewString(),
					OrderID:         orderID,
					BankAccountID:   bankAccount.ID,
					Amount:          paidAmount,
					PaymentType:     "dp",
					Status:          string(domain.PaymentVerificationVerified),
					VerifiedByID:    &verifierID,
					VerifiedAt:      &payTime,
					PaymentDate:     payTime,
					ReferenceNumber: fmt.Sprintf("SEED-PAY-%04d", orderCounter),
				}
				h.db.Create(&payment)
			}
			orderCounter++
		}

		if rand.Intn(100) < 60 {
			expCat := expenseCats[rand.Intn(len(expenseCats))]
			var poIDPtr *string
			var expAmount float64

			if expCat.Type == "hpp" {
				poIDPtr = activePoID
				expAmount = float64(rand.Intn(3000)*1000 + 500000)
			} else {
				poIDPtr = nil
				expAmount = float64(rand.Intn(200)*1000 + 50000)
			}

			expense := postgresRepo.ExpenseModel{
				ID:                uuid.NewString(),
				ExpenseCategoryID: expCat.ID,
				BatchPoID:         poIDPtr,
				Title:             fmt.Sprintf("Pengeluaran Dummy - %s", expCat.Name),
				Amount:            expAmount,
				ExpenseDate:       d.Add(14 * time.Hour),
				Notes:             "Dibuat otomatis oleh Seeder SIKOn",
				CreatedByID:       activeSalesIDs[rand.Intn(len(activeSalesIDs))],
			}
			h.db.Create(&expense)
			expenseCounter++
		}
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menyuntikkan data Seeder secara lengkap (Products, Fabrics, Designer Models, Orders, Payments, & Expenses)!", fiber.Map{
		"total_orders_generated":   orderCounter - 1,
		"total_expenses_generated": expenseCounter,
	})
}

func pointerInt(v int) *int {
	return &v
}
