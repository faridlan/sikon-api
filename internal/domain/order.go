package domain

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"
)

type OrderStatus string
type PaymentStatus string

const (
	OrderStatusQuotation  OrderStatus = "quotation"
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProduction OrderStatus = "production"
	OrderStatusReady      OrderStatus = "ready"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCanceled   OrderStatus = "canceled"

	PaymentStatusUnpaid  PaymentStatus = "unpaid"
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPartial PaymentStatus = "partial"
	PaymentStatusPaid    PaymentStatus = "paid"
)

type OrderItem struct {
	ID         string
	OrderID    string
	ProductID  string
	CustomName string
	Qty        int
	Price      float64
	Details    any
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Product *Product
}

type Order struct {
	ID                string
	OrderNumber       string
	BatchPoID         string
	BatchPO           *BatchPO
	CustomerID        string
	SalesID           string
	Subtotal          float64
	DiscountAmount    float64
	IsTaxable         bool    // 👈 Flag apakah order menggunakan PPN & PPh 22
	TaxPpnRate        float64 // 👈 Default 12.00 (%)
	TaxPph22Rate      float64 // 👈 Default 1.50 (%)
	DppPpn            float64 // 👈 Subtotal / 1.09 (atau DPP PPN Instansi)
	DppPph            float64 // 👈 Dasar Pengenaan PPh (Subtotal)
	TaxPpn            float64 // 👈 Hasil PPN 12%
	TaxPph            float64 // 👈 Hasil PPh 22 (1.5%)
	TotalAmount       float64 // 👈 Real Nilai Barang (Grand Total)
	PaguAmount        float64 // 👈 Grand Total + PPN (Invoice Gross ke Dinas)
	NetReceivedAmount float64 // 👈 Pagu - (PPN + PPh 22) / Nilai Bersih Cair ke Penyedia
	TotalQty          int
	ShippingCost      float64
	CourierName       string
	ShippingAddress   string
	OrderStatus       OrderStatus
	PaymentStatus     PaymentStatus
	ValidUntil        *time.Time
	TermsConditions   string
	Notes             string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ApprovedAt        *time.Time
	HPPMaterialCost   float64
	HPPCalculatedAt   *time.Time

	// Relasi
	Items    []OrderItem
	Customer *Customer
	Sales    *User
}

type OrderItemInput struct {
	ProductID  string
	CustomName string
	Qty        int
	Price      float64
	Details    any
}

type OrderCreateInput struct {
	BatchPoID       string
	CustomerID      string
	SalesID         string
	IsTaxable       bool    // 👈
	TaxPpnRate      float64 // 👈
	TaxPph22Rate    float64 // 👈
	ShippingCost    float64
	CourierName     string
	ShippingAddress string
	ValidUntil      *time.Time
	TermsConditions string
	Notes           string
	OrderStatus     OrderStatus
	Items           []OrderItemInput
}

type OrderUpdateInput struct {
	IsTaxable       bool    // 👈
	TaxPpnRate      float64 // 👈
	TaxPph22Rate    float64 // 👈
	ShippingCost    float64
	CourierName     string
	ShippingAddress string
	ValidUntil      *time.Time
	TermsConditions string
	Notes           string
}

type OrderFilter struct {
	Search        string
	BatchPoID     string
	CustomerID    string
	SalesID       string
	OrderStatus   string
	PaymentStatus string
	StartDate     string
	EndDate       string
}

type OrderHPPBreakdown struct {
	OrderID              string
	MaterialCost         float64
	MaterialCalculatedAt *time.Time
	LaborCost            float64
	TotalCost            float64
}

type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	GetByID(ctx context.Context, id string) (*Order, error)
	Fetch(ctx context.Context, filter OrderFilter, limit, offset int) ([]Order, int64, error)
	Update(ctx context.Context, order *Order) error
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, orderStatus OrderStatus, paymentStatus PaymentStatus) error
	CreateItem(ctx context.Context, item *OrderItem) error
	UpdateItem(ctx context.Context, item *OrderItem) error
	DeleteItem(ctx context.Context, orderID, itemID string) error
	GetItemByID(ctx context.Context, orderID, itemID string) (*OrderItem, error)
	GetByIDForUpdate(ctx context.Context, id string) (*Order, error)
	GetByBatchPOID(ctx context.Context, batchPoID string) ([]Order, error)
}

type OrderUsecase interface {
	CreateOrder(ctx context.Context, input OrderCreateInput) (*Order, error)
	GetOrder(ctx context.Context, id string) (*Order, error)
	ListOrders(ctx context.Context, filter OrderFilter, query PaginationQuery) ([]Order, PaginationMeta, error)
	UpdateOrder(ctx context.Context, id string, input OrderUpdateInput) (*Order, error)
	DeleteOrder(ctx context.Context, id string) error
	UpdateOrderStatus(ctx context.Context, id string, status OrderStatus) error
	UpdatePaymentStatus(ctx context.Context, id string, status PaymentStatus) error
	AddOrderItem(ctx context.Context, orderID string, input OrderItemInput) (*Order, error)
	UpdateOrderItem(ctx context.Context, orderID, itemID string, input OrderItemInput) (*Order, error)
	DeleteOrderItem(ctx context.Context, orderID, itemID string) (*Order, error)
	GetOrderHPP(ctx context.Context, id string) (*OrderHPPBreakdown, error)
}

// CalculateTotals menghitung ulang matematika finansial & perpajakan pengadaan instansi
func (o *Order) CalculateTotals() {
	if len(o.Items) > 0 {
		var subtotal float64
		var totalQty int
		for _, item := range o.Items {
			subtotal += item.Price * float64(item.Qty)
			totalQty += item.Qty
		}
		o.Subtotal = subtotal
		o.TotalQty = totalQty
	}

	o.TotalAmount = (o.Subtotal - o.DiscountAmount) + o.ShippingCost

	if o.IsTaxable {
		if o.TaxPpnRate <= 0 {
			o.TaxPpnRate = 12.00
		}
		if o.TaxPph22Rate <= 0 {
			o.TaxPph22Rate = 1.50
		}

		// 🚨 RUMUS INSTANSI DINAS:
		// 1. DPP PPN dihitung via 100/109 (atau 4.881.250 untuk nilai 5.325.000)
		o.DppPpn = math.Floor((o.TotalAmount * (100.0 / 109.0))) // Menghasilkan 4.885.321
		o.DppPph = o.TotalAmount                                 // DPP PPh = Nilai Real Barang (5.325.000)

		// 2. PPN 12% = 11% dari Nilai Real (atau 12% dari DPP Instansi 4.881.250 -> 585.750)
		// Rumus presisi dinas: Subtotal * (11 / 100)
		o.TaxPpn = math.Round(o.TotalAmount * 0.11) // 5.325.000 * 11% = 585.750

		// 3. PPh 22 (1.5%) dihitung dari DPP PPH (Subtotal Real Barang)
		o.TaxPph = math.Round(o.DppPph * (o.TaxPph22Rate / 100)) // 5.325.000 * 1,5% = 79.875

		// 4. Pagu Belanja (Invoice Gross) = Subtotal + PPN
		o.PaguAmount = o.TotalAmount + o.TaxPpn // 5.325.000 + 585.750 = 5.910.750

		// 5. Yang Diterima Penyedia = Pagu - (PPN + PPh 22)
		o.NetReceivedAmount = o.PaguAmount - (o.TaxPpn + o.TaxPph) // 5.245.125
	} else {
		o.DppPpn = 0
		o.DppPph = 0
		o.TaxPpn = 0
		o.TaxPph = 0
		o.PaguAmount = o.TotalAmount
		o.NetReceivedAmount = o.TotalAmount
	}
}

func (o *Order) GenerateOrderNumber() error {
	dateStr := time.Now().Format("20060102")
	max := big.NewInt(100000)
	randNum, err := rand.Int(rand.Reader, max)
	if err != nil {
		return err
	}

	o.OrderNumber = fmt.Sprintf("ORD-%s-%05d", dateStr, randNum.Int64())
	return nil
}

func (o *Order) TransitionStatus(newStatus OrderStatus) error {
	if o.OrderStatus == newStatus {
		return nil
	}

	if o.OrderStatus == OrderStatusCompleted || o.OrderStatus == OrderStatusCanceled {
		return NewError(ErrConflict, "Pesanan yang sudah selesai atau dibatalkan tidak dapat diubah statusnya")
	}

	switch newStatus {
	case OrderStatusPending:
		if o.OrderStatus != OrderStatusQuotation && o.OrderStatus != OrderStatusProduction {
			return NewError(ErrConflict, "Hanya pesanan berstatus quotation atau production (koreksi) yang bisa diubah menjadi pending")
		}
		if o.PaymentStatus == PaymentStatusUnpaid {
			return NewError(ErrConflict, "Pesanan tidak dapat diproses (pending) karena belum ada pembayaran (minimal DP)")
		}

	case OrderStatusProduction:
		if o.OrderStatus != OrderStatusPending && o.OrderStatus != OrderStatusReady {
			return NewError(ErrConflict, "Pesanan harus berstatus pending (antrean produksi) sebelum masuk meja produksi")
		}
		if o.PaymentStatus == PaymentStatusUnpaid || o.PaymentStatus == PaymentStatusPending {
			return NewError(ErrConflict, "Pesanan tidak dapat masuk produksi karena pembayaran DP belum diverifikasi oleh Accounting")
		}

	case OrderStatusReady:
		if o.OrderStatus != OrderStatusProduction {
			return NewError(ErrConflict, "Pesanan harus berstatus production sebelum dipindahkan ke barang jadi (ready)")
		}

	case OrderStatusCompleted:
		if o.OrderStatus != OrderStatusReady {
			return NewError(ErrConflict, "Pesanan belum siap (ready), tidak bisa diselesaikan")
		}
		if o.PaymentStatus != PaymentStatusPaid {
			return NewError(ErrConflict, "Pesanan tidak dapat diselesaikan/diambil customer karena belum lunas sepenuhnya (atau pelunasan belum diverifikasi Accounting)")
		}

	case OrderStatusCanceled:
		if o.OrderStatus == OrderStatusProduction || o.OrderStatus == OrderStatusReady {
			return NewError(ErrConflict, "Pesanan yang sedang/sudah diproduksi tidak dapat dibatalkan begitu saja")
		}

	default:
		return NewError(ErrBadParamInput, "Status transisi tidak dikenali")
	}

	o.OrderStatus = newStatus
	return nil
}

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusQuotation, OrderStatusPending, OrderStatusProduction, OrderStatusReady, OrderStatusCompleted, OrderStatusCanceled:
		return true
	}
	return false
}

func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusUnpaid, PaymentStatusPartial, PaymentStatusPaid:
		return true
	}
	return false
}
