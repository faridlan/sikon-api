package domain

import (
	"context"
	"crypto/rand"
	"fmt"
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
	// Details untuk menyimpan JSON variasi (misal: S: 10, M: 20)
	Details   any
	CreatedAt time.Time
	UpdatedAt time.Time

	Product *Product
}

type Order struct {
	ID              string
	OrderNumber     string
	BatchPoID       string
	BatchPO         *BatchPO
	CustomerID      string
	SalesID         string
	Subtotal        float64
	DiscountAmount  float64
	TaxPpn          float64
	TaxPph          float64
	TotalAmount     float64
	ShippingCost    float64
	CourierName     string
	ShippingAddress string
	OrderStatus     OrderStatus
	PaymentStatus   PaymentStatus
	ValidUntil      *time.Time
	TermsConditions string
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ApprovedAt      *time.Time

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
	ShippingCost    float64
	CourierName     string
	ShippingAddress string
	ValidUntil      *time.Time
	TermsConditions string
	Notes           string
}

type OrderFilter struct {
	Search        string // Untuk pencarian OrderNumber
	BatchPoID     string
	CustomerID    string // Filter by Customer
	SalesID       string // Filter by Sales
	OrderStatus   string // Filter by Order Status
	PaymentStatus string // Filter by Payment Status
	StartDate     string // Filter dari tanggal (format: YYYY-MM-DD)
	EndDate       string // Filter sampai tanggal (format: YYYY-MM-DD)
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
}

// CalculateTotals menghitung ulang Subtotal dan TotalAmount pesanan.
// Sesuai prinsip DRY, logika matematika finansial terpusat di sini.
func (o *Order) CalculateTotals() {
	// 1. Hitung ulang subtotal HANYA JIKA array Items tersedia di memory
	// (Mencegah subtotal jadi 0 jika kita hanya me-load header order dari DB)
	if len(o.Items) > 0 {
		var subtotal float64
		for _, item := range o.Items {
			subtotal += item.Price * float64(item.Qty)
		}
		o.Subtotal = subtotal
	}

	// 2. Hitung Grand Total
	o.TotalAmount = (o.Subtotal - o.DiscountAmount) + o.TaxPpn - o.TaxPph + o.ShippingCost
}

// GenerateOrderNumber membuat nomor pesanan unik dengan format ORD-YYYYMMDD-XXXXX.
// Menggunakan crypto/rand untuk mencegah tabrakan data (collision) saat sistem berjalan paralel.
func (o *Order) GenerateOrderNumber() error {
	dateStr := time.Now().Format("20060102")

	// Menghasilkan angka acak dari 0 hingga 99999 dengan aman
	max := big.NewInt(100000)
	randNum, err := rand.Int(rand.Reader, max)
	if err != nil {
		return err
	}

	o.OrderNumber = fmt.Sprintf("ORD-%s-%05d", dateStr, randNum.Int64())
	return nil
}

// TransitionStatus memvalidasi dan mengubah status order berdasarkan alur pabrik.
func (o *Order) TransitionStatus(newStatus OrderStatus) error {
	// 1. Jika status yang diminta sama dengan status saat ini, abaikan saja
	if o.OrderStatus == newStatus {
		return nil
	}

	// 2. Jika order sudah berstatus Final (Selesai/Batal), tidak boleh diutak-atik lagi (Terminal State)
	if o.OrderStatus == OrderStatusCompleted || o.OrderStatus == OrderStatusCanceled {
		return NewError(ErrConflict, "Pesanan yang sudah selesai atau dibatalkan tidak dapat diubah statusnya")
	}

	// 3. Evaluasi Aturan Transisi (State Machine)
	switch newStatus {

	case OrderStatusPending:
		// MVP REVISI: Izinkan kembali ke pending dari production jika admin salah klik (koreksi manual)
		if o.OrderStatus != OrderStatusQuotation && o.OrderStatus != OrderStatusProduction {
			return NewError(ErrConflict, "Hanya pesanan berstatus quotation atau production (koreksi) yang bisa diubah menjadi pending")
		}
		// 🚨 GEMBOK UANG 1: Harus ada uang masuk (DP/Lunas) sebelum masuk jadwal pabrik
		if o.PaymentStatus == PaymentStatusUnpaid {
			return NewError(ErrConflict, "Pesanan tidak dapat diproses (pending) karena belum ada pembayaran (minimal DP)")
		}

	case OrderStatusProduction:
		// Syarat: Harus dari meja Pending (atau koreksi dari Ready jika barang ternyata cacat dan harus dijahit ulang)
		if o.OrderStatus != OrderStatusPending && o.OrderStatus != OrderStatusReady {
			return NewError(ErrConflict, "Pesanan harus berstatus pending sebelum masuk meja produksi")
		}

	case OrderStatusReady:
		// Syarat: Baju harus sudah selesai dijahit
		if o.OrderStatus != OrderStatusProduction {
			return NewError(ErrConflict, "Pesanan harus berstatus production sebelum bisa dipindahkan ke gudang (ready)")
		}
		// Catatan: Di titik ini uang BOLEH masih partial. Ini justru waktu yang tepat untuk menagih customer!

	case OrderStatusCompleted:
		// Syarat mutlak posisi barang: Harus dari gudang barang jadi (Ready)
		if o.OrderStatus != OrderStatusReady {
			return NewError(ErrConflict, "Pesanan belum siap (ready), tidak bisa diselesaikan")
		}
		// 🚨 GEMBOK UANG 2: Wajib Lunas sebelum diserahkan/diambil customer
		if o.PaymentStatus != PaymentStatusPaid {
			return NewError(ErrConflict, "Pesanan tidak dapat diselesaikan karena belum lunas sepenuhnya")
		}

	case OrderStatusCanceled:
		// Syarat: Jika sudah masuk pabrik atau sudah jadi, tidak boleh dibatalkan sembarangan!
		if o.OrderStatus == OrderStatusProduction || o.OrderStatus == OrderStatusReady {
			return NewError(ErrConflict, "Pesanan yang sedang/sudah diproduksi tidak dapat dibatalkan begitu saja")
		}

	default:
		return NewError(ErrBadParamInput, "Status transisi tidak dikenali")
	}

	// 4. Jika lolos semua pemeriksaan satpam, setujui status barunya
	o.OrderStatus = newStatus
	return nil
}

// Validasi untuk OrderStatus
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusQuotation, OrderStatusPending, OrderStatusProduction, OrderStatusReady, OrderStatusCompleted, OrderStatusCanceled:
		return true
	}
	return false
}

// Validasi untuk PaymentStatus
func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusUnpaid, PaymentStatusPartial, PaymentStatusPaid:
		return true
	}
	return false
}
