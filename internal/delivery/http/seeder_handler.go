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
	// Hard-delete data dummy (menggunakan wildcard yang mencakup semua dummy sales)
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

	// A. Buat 3 User Sales
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

	// B. Buat Bank Account Dummy
	bankAccount := postgresRepo.BankAccountModel{
		ID:            uuid.NewString(),
		BankName:      "BCA",
		AccountNumber: "999888777",
		AccountName:   "PT Konveksi Dummy",
	}
	h.db.Create(&bankAccount)

	// C. Buat Kategori & Produk
	cat := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Dummy Cat Kemeja"}
	h.db.Create(&cat)

	product1 := postgresRepo.ProductModel{ID: uuid.NewString(), CategoryID: cat.ID, Name: "Dummy Prod PDL", BasePrice: 150000}
	product2 := postgresRepo.ProductModel{ID: uuid.NewString(), CategoryID: cat.ID, Name: "Dummy Prod PDH", BasePrice: 180000}
	h.db.Create(&product1)
	h.db.Create(&product2)
	products := []postgresRepo.ProductModel{product1, product2}

	// D. Buat Customers dan distribusikan secara acak ke para Sales
	var customers []postgresRepo.CustomerModel
	customerNames := []string{"Dummy Cust PT A", "Dummy Cust PT B", "Dummy Cust Personal C", "Dummy Cust CV D", "Dummy Cust Personal E"}

	for i, name := range customerNames {
		// Ambil random ID Sales
		randomSalesID := activeSalesIDs[rand.Intn(len(activeSalesIDs))]

		cust := postgresRepo.CustomerModel{
			ID:        uuid.NewString(),
			Name:      name,
			Phone:     fmt.Sprintf("08111222%d", i),
			CreatedBy: randomSalesID,
			SalesID:   &randomSalesID, // Di-assign ke sales terpilih
		}
		h.db.Create(&cust)
		customers = append(customers, cust)
	}

	// --- 2. SETUP BATCH PO (Juni - Agustus) ---
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
		{"PO 1 AGUSTUS 2026", 8, 2026, "2026-07-25", "2026-08-01", string(domain.BatchPOStatusActive)},
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
	endDate, _ := time.Parse("2006-01-02", "2026-08-01")

	orderCounter := 1

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

		// Simulasi pembuatan 1 - 3 Order per hari
		numOrders := rand.Intn(3) + 1
		for i := 0; i < numOrders; i++ {
			// Pilih customer secara acak
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
				SalesID:       *cust.SalesID, // Pakai SalesID yang terikat di Customer!
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
				payment := postgresRepo.PaymentModel{
					ID:              uuid.NewString(),
					OrderID:         orderID,
					BankAccountID:   bankAccount.ID,
					Amount:          paidAmount,
					PaymentType:     "dp",
					PaymentDate:     approvedTime.Add(2 * time.Hour),
					ReferenceNumber: fmt.Sprintf("SEED-PAY-%04d", orderCounter),
				}
				h.db.Create(&payment)
			}
			orderCounter++
		}
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menyuntikkan data Seeder dengan Multi-Sales secara realistis!", fiber.Map{
		"total_orders_generated": orderCounter - 1,
	})
}
