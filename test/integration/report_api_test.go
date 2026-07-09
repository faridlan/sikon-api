package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/domain"
	postgresRepo "github.com/faridlan/sikon-api/internal/repository/postgres"
	tests "github.com/faridlan/sikon-api/test"
)

func TestReportDailyEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := postgresRepo.CategoryModel{
		ID:   uuid.NewString(),
		Name: "Atasan",
	}
	assert.NoError(t, db.Create(&category).Error)

	product := postgresRepo.ProductModel{
		ID:         uuid.NewString(),
		CategoryID: category.ID,
		Name:       "Kemeja",
		BasePrice:  100000,
	}
	assert.NoError(t, db.Create(&product).Error)

	sales := postgresRepo.UserModel{
		ID:       uuid.NewString(),
		Name:     "Rina",
		Email:    "rina@example.com",
		Password: "secret",
		Role:     "sales",
	}
	assert.NoError(t, db.Create(&sales).Error)

	customer := postgresRepo.CustomerModel{
		ID:        uuid.NewString(),
		Name:      "PT Maju",
		Phone:     "0812",
		Address:   "Bandung",
		CreatedBy: sales.ID,
		SalesID:   &sales.ID,
	}
	assert.NoError(t, db.Create(&customer).Error)

	batchPO := postgresRepo.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO-001",
		Status:    string(domain.BatchPOStatusActive),
		Quota:     10,
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(24 * time.Hour),
	}
	assert.NoError(t, db.Create(&batchPO).Error)

	order := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-TEST-001",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   300000,
		Subtotal:      300000,
		OrderStatus:   string(domain.OrderStatusProduction),
		PaymentStatus: string(domain.PaymentStatusUnpaid),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	assert.NoError(t, db.Create(&order).Error)

	orderItem := postgresRepo.OrderItemModel{
		ID:        uuid.NewString(),
		OrderID:   order.ID,
		ProductID: product.ID,
		Qty:       3,
		Price:     100000,
	}
	assert.NoError(t, db.Create(&orderItem).Error)

	req := httptest.NewRequest("GET", "/api/reports/daily?date="+time.Now().Format("2006-01-02"), bytes.NewBuffer(nil))
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response struct {
		Data struct {
			DailySnapshot struct {
				TotalRevenueToday float64 `json:"total_revenue_today"`
				TotalQtyToday     int64   `json:"total_qty_today"`
			} `json:"daily_snapshot"`
			ActivePOs []struct {
				BatchPOName string `json:"batch_po_name"`
			} `json:"active_pos"`
			TotalOutstandingReceivables float64 `json:"total_outstanding_receivables"`
			ActivePOReceivables         float64 `json:"active_po_receivables"`
		} `json:"data"`
	}

	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	assert.Equal(t, float64(300000), response.Data.DailySnapshot.TotalRevenueToday)
	assert.Equal(t, int64(3), response.Data.DailySnapshot.TotalQtyToday)
	assert.NotEmpty(t, response.Data.ActivePOs)
	assert.GreaterOrEqual(t, response.Data.TotalOutstandingReceivables, float64(0))
}
