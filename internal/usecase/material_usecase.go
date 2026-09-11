package usecase

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// ==========================================
// USECASE: MATERIAL
// ==========================================
type materialUsecase struct {
	materialRepo   domain.MaterialRepository
	contextTimeout time.Duration
}

func NewMaterialUsecase(repo domain.MaterialRepository, timeout time.Duration) domain.MaterialUsecase {
	return &materialUsecase{
		materialRepo:   repo,
		contextTimeout: timeout,
	}
}

func (u *materialUsecase) CreateMaterial(c context.Context, input domain.MaterialCreateInput) (*domain.Material, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.Name == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Nama material tidak boleh kosong")
	}
	if input.Unit == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Satuan material harus diisi")
	}
	if input.UnitPrice <= 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Harga satuan harus lebih besar dari 0")
	}

	material := &domain.Material{
		Name:      input.Name,
		Unit:      input.Unit,
		UnitPrice: input.UnitPrice,
		Category:  input.Category,
	}

	if err := u.materialRepo.Create(ctx, material); err != nil {
		return nil, err
	}

	return material, nil
}

func (u *materialUsecase) GetMaterial(c context.Context, id string) (*domain.Material, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID Material tidak valid")
	}

	return u.materialRepo.GetByID(ctx, id)
}

func (u *materialUsecase) ListMaterials(c context.Context, query domain.PaginationQuery) ([]domain.Material, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	data, total, err := u.materialRepo.Fetch(ctx, query.Limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       query.Limit,
		TotalItems:  total,
		TotalPages:  int(math.Ceil(float64(total) / float64(query.Limit))),
	}

	return data, meta, nil
}

func (u *materialUsecase) UpdateMaterial(c context.Context, id string, input domain.MaterialUpdateInput) (*domain.Material, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existing, err := u.materialRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		existing.Name = input.Name
	}
	if input.Unit != "" {
		existing.Unit = input.Unit
	}
	if input.UnitPrice > 0 {
		existing.UnitPrice = input.UnitPrice
	}
	if input.Category != "" {
		existing.Category = input.Category
	}

	if err := u.materialRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (u *materialUsecase) DeleteMaterial(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return domain.NewError(domain.ErrBadParamInput, "ID tidak valid")
	}

	return u.materialRepo.Delete(ctx, id)
}

// ==========================================
// USECASE: PRODUCT MATERIAL (Resep / BOM)
// ==========================================
type productMaterialUsecase struct {
	productMaterialRepo domain.ProductMaterialRepository
	productRepo         domain.ProductRepository
	txManager           domain.TransactionManager
	contextTimeout      time.Duration
}

func NewProductMaterialUsecase(
	pmRepo domain.ProductMaterialRepository,
	productRepo domain.ProductRepository,
	txManager domain.TransactionManager,
	timeout time.Duration,
) domain.ProductMaterialUsecase {
	return &productMaterialUsecase{
		productMaterialRepo: pmRepo,
		productRepo:         productRepo,
		txManager:           txManager,
		contextTimeout:      timeout,
	}
}

func (u *productMaterialUsecase) SetProductMaterials(c context.Context, input domain.SetProductMaterialsInput) ([]domain.ProductMaterial, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.ProductID == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Product ID tidak valid")
	}

	// Pastikan produknya ada
	if _, err := u.productRepo.GetByID(ctx, input.ProductID); err != nil {
		return nil, err
	}

	items := make([]domain.ProductMaterial, len(input.Items))
	for i, it := range input.Items {
		if it.MaterialID == "" {
			return nil, domain.NewError(domain.ErrBadParamInput, "Material ID pada baris resep tidak boleh kosong")
		}
		if it.QtyPerUnit <= 0 {
			return nil, domain.NewError(domain.ErrBadParamInput, "Qty per unit harus lebih besar dari 0")
		}
		items[i] = domain.ProductMaterial{
			ProductID:  input.ProductID,
			MaterialID: it.MaterialID,
			QtyPerUnit: it.QtyPerUnit,
		}
	}

	err := u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		return u.productMaterialRepo.ReplaceForProduct(txCtx, input.ProductID, items)
	})
	if err != nil {
		return nil, err
	}

	return u.productMaterialRepo.FetchByProduct(ctx, input.ProductID)
}

func (u *productMaterialUsecase) GetProductMaterials(c context.Context, productID string) ([]domain.ProductMaterial, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if productID == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Product ID tidak valid")
	}

	return u.productMaterialRepo.FetchByProduct(ctx, productID)
}

// CalculateMaterialCost: qty pesanan dikali seluruh resep produk ini.
// Ini fungsi yang nanti dipanggil order_usecase.go saat membuat/approve Order.
func (u *productMaterialUsecase) CalculateMaterialCost(c context.Context, productID string, qty int) (float64, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	items, err := u.productMaterialRepo.FetchByProduct(ctx, productID)
	if err != nil {
		slog.Error("[HPP-DEBUG] FetchByProduct error", "product_id", productID, "error", err)
		return 0, err
	}
	slog.Debug("[HPP-DEBUG] FetchByProduct hasil", "product_id", productID, "jumlah_baris_resep", len(items))

	var total float64
	for _, item := range items {
		if item.Material == nil {
			slog.Warn("[HPP-DEBUG] Material NIL pada baris resep", "material_id", item.MaterialID)
			continue
		}
		sub := item.QtyPerUnit * item.Material.UnitPrice * float64(qty)
		slog.Debug("[HPP-DEBUG] baris resep", "material", item.Material.Name, "qty_per_unit", item.QtyPerUnit, "unit_price", item.Material.UnitPrice, "qty_order", qty, "subtotal", sub)
		total += sub
	}

	slog.Debug("[HPP-DEBUG] CalculateMaterialCost selesai", "product_id", productID, "qty", qty, "total", total)
	return total, nil
}
