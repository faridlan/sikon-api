package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type productUsecase struct {
	productRepo    domain.ProductRepository
	categoryRepo   domain.CategoryRepository
	storageService domain.StorageService
	txManager      domain.TransactionManager
	contextTimeout time.Duration
}

func NewProductUsecase(pr domain.ProductRepository, cr domain.CategoryRepository, ss domain.StorageService, tm domain.TransactionManager, timeout time.Duration) domain.ProductUsecase {
	return &productUsecase{
		productRepo:    pr,
		categoryRepo:   cr,
		storageService: ss,
		txManager:      tm,
		contextTimeout: timeout,
	}
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}

func (u *productUsecase) CreateProduct(c context.Context, input domain.ProductCreateInput) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Validasi CategoryID
	_, err := u.categoryRepo.GetByID(ctx, input.CategoryID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrBadParamInput, "Kategori tidak ditemukan")
		}
		return nil, err
	}

	// 2. Slug Handling
	slug := input.Slug
	if slug == "" {
		slug = generateSlug(input.Name)
	}

	// Pastikan Slug unik
	if existing, _ := u.productRepo.GetBySlug(ctx, slug); existing != nil {
		slug = fmt.Sprintf("%s-%d", slug, time.Now().Unix())
	}

	product := &domain.Product{
		CategoryID:    input.CategoryID,
		Name:          input.Name,
		Description:   input.Description,
		BasePrice:     input.BasePrice,
		Slug:          slug,
		GSMInfo:       input.GSMInfo,
		FabricSummary: input.FabricSummary,
		KeyFeatures:   input.KeyFeatures,
		Rating:        0.0,
		SoldCount:     0,
		ReviewCount:   0,
	}

	// 3. Mapping Product Images
	if len(input.ImageURLs) > 0 {
		var images []domain.ProductImage
		for i, url := range input.ImageURLs {
			images = append(images, domain.ProductImage{
				ImageURL:  url,
				IsPrimary: i == 0,
			})
		}
		product.Images = images
	}

	// 4. Mapping Product Fabrics & Colors
	if len(input.Fabrics) > 0 {
		var fabrics []domain.ProductFabric
		for _, fabInput := range input.Fabrics {
			var colors []domain.FabricColor
			for _, cInput := range fabInput.Colors {
				colors = append(colors, domain.FabricColor{
					Name:    cInput.Name,
					HexCode: cInput.HexCode,
				})
			}
			qty := fabInput.QtyPerUnit
			if qty <= 0 {
				qty = 1.5
			}
			fabrics = append(fabrics, domain.ProductFabric{
				FabricID:        fabInput.FabricID,
				QtyPerUnit:      qty,
				SpecTemplateID:  fabInput.SpecTemplateID,
				Name:            fabInput.Name,
				Description:     fabInput.Description,
				Composition:     fabInput.Composition,
				CareInstruction: fabInput.CareInstruction,
				BasePrice:       fabInput.BasePrice,
				PriceAdjustment: fabInput.PriceAdjustment,
				IsDefault:       fabInput.IsDefault,
				Colors:          colors,
			})
		}
		product.Fabrics = fabrics
	}

	// 5. Mapping Wholesale Prices
	if len(input.Wholesale) > 0 {
		var wholesales []domain.WholesalePrice
		for _, wInput := range input.Wholesale {
			wholesales = append(wholesales, domain.WholesalePrice{
				FabricID:  wInput.FabricID,
				MinQty:    wInput.MinQty,
				MaxQty:    wInput.MaxQty,
				UnitPrice: wInput.UnitPrice,
			})
		}
		product.Wholesale = wholesales
	}

	// 6. Mapping Designer Model
	if input.DesignModel != nil {
		var views []domain.ProductModelView
		for _, vInput := range input.DesignModel.Views {
			views = append(views, domain.ProductModelView{
				Side:    vInput.Side,
				ArtURL:  vInput.ArtURL,
				MaskURL: vInput.MaskURL,
				Width:   vInput.Width,
				Height:  vInput.Height,
			})
		}
		product.DesignModel = &domain.ProductModel{
			Name:        input.DesignModel.Name,
			Type:        input.DesignModel.Type,
			Description: input.DesignModel.Description,
			Views:       views,
		}
	}

	// Execute Create dalam Transaction
	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		return u.productRepo.Create(txCtx, product)
	})
	if err != nil {
		return nil, err
	}

	// Reload dari DB agar relasi (Fabric, Colors, Category) ter-preload penuh di response
	reloaded, err := u.productRepo.GetByID(ctx, product.ID)
	if err != nil {
		return product, nil // fallback ke data in-memory jika reload gagal
	}

	return reloaded, nil
}

func (u *productUsecase) GetProduct(c context.Context, id string) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	product, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Produk dengan ID tersebut tidak ditemukan")
		}
		return nil, err
	}

	return product, nil
}

func (u *productUsecase) GetProductBySlug(c context.Context, slug string) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	product, err := u.productRepo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Produk dengan slug tersebut tidak ditemukan")
		}
		return nil, err
	}

	return product, nil
}

func (u *productUsecase) ListProducts(c context.Context, filter domain.ProductFilter, query domain.PaginationQuery) ([]domain.Product, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	products, totalItems, err := u.productRepo.Fetch(ctx, filter, limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	return products, meta, nil
}

func (u *productUsecase) UpdateProduct(c context.Context, id string, input domain.ProductUpdateInput) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingProduct, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Produk dengan ID tersebut tidak ditemukan")
		}
		return nil, err
	}

	if input.CategoryID != "" && input.CategoryID != existingProduct.CategoryID {
		_, err := u.categoryRepo.GetByID(ctx, input.CategoryID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.NewError(domain.ErrBadParamInput, "Kategori baru tidak ditemukan")
			}
			return nil, err
		}
		existingProduct.CategoryID = input.CategoryID
	}

	if input.Name != "" {
		existingProduct.Name = input.Name
	}
	if input.Description != "" {
		existingProduct.Description = input.Description
	}
	if input.BasePrice > 0 {
		existingProduct.BasePrice = input.BasePrice
	}
	if input.Slug != "" {
		existingProduct.Slug = input.Slug
	}
	if input.GSMInfo != "" {
		existingProduct.GSMInfo = input.GSMInfo
	}
	if input.FabricSummary != "" {
		existingProduct.FabricSummary = input.FabricSummary
	}
	if input.KeyFeatures != nil {
		existingProduct.KeyFeatures = input.KeyFeatures
	}

	var urlsToDelete []string

	// Sync Images
	if input.ImageURLs != nil {
		oldImagesMap := make(map[string]string)
		for _, img := range existingProduct.Images {
			oldImagesMap[img.ImageURL] = img.ID
		}

		var newImages []domain.ProductImage
		for i, url := range input.ImageURLs {
			imageID := oldImagesMap[url]
			newImages = append(newImages, domain.ProductImage{
				ID:        imageID,
				ImageURL:  url,
				IsPrimary: i == 0,
			})
			delete(oldImagesMap, url)
		}

		for url := range oldImagesMap {
			urlsToDelete = append(urlsToDelete, url)
		}

		existingProduct.Images = newImages
	}

	// Update Fabrics
	if input.Fabrics != nil {
		var fabrics []domain.ProductFabric
		for _, fabInput := range input.Fabrics {
			var colors []domain.FabricColor
			for _, cInput := range fabInput.Colors {
				colors = append(colors, domain.FabricColor{
					Name:    cInput.Name,
					HexCode: cInput.HexCode,
				})
			}
			qty := fabInput.QtyPerUnit
			if qty <= 0 {
				qty = 1.5
			}
			fabrics = append(fabrics, domain.ProductFabric{
				FabricID:        fabInput.FabricID,
				QtyPerUnit:      qty,
				SpecTemplateID:  fabInput.SpecTemplateID,
				Name:            fabInput.Name,
				Description:     fabInput.Description,
				Composition:     fabInput.Composition,
				CareInstruction: fabInput.CareInstruction,
				BasePrice:       fabInput.BasePrice,
				PriceAdjustment: fabInput.PriceAdjustment,
				IsDefault:       fabInput.IsDefault,
				Colors:          colors,
			})
		}
		existingProduct.Fabrics = fabrics
	}

	// Update Wholesale
	if input.Wholesale != nil {
		var wholesales []domain.WholesalePrice
		for _, wInput := range input.Wholesale {
			wholesales = append(wholesales, domain.WholesalePrice{
				FabricID:  wInput.FabricID,
				MinQty:    wInput.MinQty,
				MaxQty:    wInput.MaxQty,
				UnitPrice: wInput.UnitPrice,
			})
		}
		existingProduct.Wholesale = wholesales
	}

	// Update Designer Model
	if input.DesignModel != nil {
		var views []domain.ProductModelView
		for _, vInput := range input.DesignModel.Views {
			views = append(views, domain.ProductModelView{
				Side:    vInput.Side,
				ArtURL:  vInput.ArtURL,
				MaskURL: vInput.MaskURL,
				Width:   vInput.Width,
				Height:  vInput.Height,
			})
		}
		existingProduct.DesignModel = &domain.ProductModel{
			Name:        input.DesignModel.Name,
			Type:        input.DesignModel.Type,
			Description: input.DesignModel.Description,
			Views:       views,
		}
	}

	// Execute Update dalam Transaction
	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		return u.productRepo.Update(txCtx, existingProduct)
	})
	if err != nil {
		return nil, err
	}

	// Hapus file di Supabase Storage di background
	if len(urlsToDelete) > 0 {
		go func(urls []string) {
			for _, urlToDelete := range urls {
				_ = u.storageService.DeleteFile(context.Background(), urlToDelete)
			}
		}(urlsToDelete)
	}

	return existingProduct, nil
}

func (u *productUsecase) DeleteProduct(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingProduct, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Produk dengan ID tersebut tidak ditemukan")
		}
		return err
	}

	if err := u.productRepo.Delete(ctx, id); err != nil {
		return err
	}

	var urlsToDelete []string
	for _, img := range existingProduct.Images {
		urlsToDelete = append(urlsToDelete, img.ImageURL)
	}
	if existingProduct.DesignModel != nil {
		for _, v := range existingProduct.DesignModel.Views {
			urlsToDelete = append(urlsToDelete, v.ArtURL, v.MaskURL)
		}
	}

	if len(urlsToDelete) > 0 {
		go func(urls []string) {
			for _, urlToDelete := range urls {
				_ = u.storageService.DeleteFile(context.Background(), urlToDelete)
			}
		}(urlsToDelete)
	}

	return nil
}
