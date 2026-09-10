package http

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	postgresRepo "github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type SeederHandler interface {
	Generate(c *fiber.Ctx) error
	Clear(c *fiber.Ctx) error
}

type seederHandler struct {
	db             *gorm.DB
	storageService domain.StorageService
}

func NewSeederHandler(db *gorm.DB, storage domain.StorageService) SeederHandler {
	return &seederHandler{
		db:             db,
		storageService: storage,
	}
}

// Helper lokal untuk membaca file dari folder lokal dan mengunggahnya ke Supabase
func (h *seederHandler) uploadLocalFile(ctx context.Context, localFilePath string, targetFolder string) (string, error) {
	// 1. Cek & Buka file fisik dari disk
	resolvedPath := localFilePath
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		resolvedPath = filepath.Join("..", "..", localFilePath)
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			return "", fmt.Errorf("file tidak ditemukan di [%s]", localFilePath)
		}
	}

	fileBytes, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("gagal membaca file %s: %w", resolvedPath, err)
	}

	ext := filepath.Ext(resolvedPath)
	filename := filepath.Base(resolvedPath)
	fileNameOnly := strings.TrimSuffix(filename, ext)
	contentType := "image/png"

	if strings.ToLower(ext) == ".jpg" || strings.ToLower(ext) == ".jpeg" {
		contentType = "image/jpeg"
	}

	// 2. Buat buffer multipart form in-memory agar fileHeader.Open() berhasil
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	hHeaders := make(textproto.MIMEHeader)
	hHeaders.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	hHeaders.Set("Content-Type", contentType)
	part, err := writer.CreatePart(hHeaders)
	if err != nil {
		return "", fmt.Errorf("gagal membuat multipart part: %w", err)
	}

	if _, err := part.Write(fileBytes); err != nil {
		return "", fmt.Errorf("gagal menulis bytes ke multipart: %w", err)
	}

	_ = writer.Close()

	// 3. Parse kembali ke multipart.Form untuk mendapatkan *multipart.FileHeader yang VALID
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20) // 10MB limit
	if err != nil {
		return "", fmt.Errorf("gagal parsing form header: %w", err)
	}

	defer form.RemoveAll()
	files := form.File["file"]
	if len(files) == 0 {
		return "", fmt.Errorf("gagal mengekstrak file header dari memory")
	}

	// 4. Upload via Storage Service
	uniqueFileName := fmt.Sprintf("seeder_%s", fileNameOnly)
	uploadedURL, err := h.storageService.UploadFile(ctx, files[0], targetFolder, uniqueFileName)
	if err != nil {
		return "", fmt.Errorf("gagal upload ke supabase: %w", err)
	}

	return uploadedURL, nil
}

// @Summary Hapus Semua Data Seeder
// @Tags Seeder
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /seeder/clear [post]
func (h *seederHandler) Clear(c *fiber.Ctx) error {

	// Hapus Relasi Produk & Resep BOM
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.ProductMaterialModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.ProductModelViewModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.DesignerModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.WholesalePriceModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.FabricColorModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.ProductFabricModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.ProductImageModel{})

	// Hapus Data Utama & Payroll Modul
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.WorkLogModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.PayrollModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.WorkerModel{})

	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.ExpenseModel{})
	h.db.Unscoped().Where("name LIKE ?", "Dummy Cat Exp%").Delete(&postgresRepo.ExpenseCategoryModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.PaymentModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.OrderItemModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.OrderModel{})
	h.db.Unscoped().Where("1=1").Delete(&postgresRepo.BatchPOModel{})
	h.db.Unscoped().Where("name LIKE ?", "Dummy Cust%").Delete(&postgresRepo.CustomerModel{}) // 👈 Hapus Customer DULU
	h.db.Unscoped().Where("name LIKE ?", "Dummy Prod%").Delete(&postgresRepo.ProductModel{})
	h.db.Unscoped().Where("name LIKE ?", "Dummy Cat%").Delete(&postgresRepo.CategoryModel{})
	h.db.Unscoped().Where("name LIKE ?", "Dummy Mat%").Delete(&postgresRepo.MaterialModel{})
	h.db.Unscoped().Where("email LIKE ?", "%_dummy@sikon.com").Delete(&postgresRepo.UserModel{}) // 👈 Hapus User SETELAH Customer
	h.db.Unscoped().Where("account_number = ?", "999888777").Delete(&postgresRepo.BankAccountModel{})
	h.db.Unscoped().Where("name LIKE ?", "Dummy Spec%").Delete(&postgresRepo.SpecTemplateModel{})

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil membersihkan data seeder", nil)
}

// @Summary Generate Data Testing Realistis
// @Tags Seeder
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /seeder/generate [post]
func (h *seederHandler) Generate(c *fiber.Ctx) error {
	ctx := c.Context()

	// Hash password dummy agar bisa digunakan untuk login
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Gagal memproses password dummy seeder")
	}

	// --- 1. SETUP MASTER DATA USER & BANK ---
	ownerDummy := postgresRepo.UserModel{
		ID:         uuid.NewString(),
		Name:       "Owner (Dummy)",
		Email:      "owner_dummy@sikon.com",
		Password:   string(hashedPassword),
		Role:       string(domain.RoleOwner),
		Phone:      "6281100000000",
		StatusText: "Online sekarang",
		IsActive:   true,
		SortOrder:  0,
	}

	var existingOwner postgresRepo.UserModel
	var ownerID string // Variable penampung ID Owner yang valid

	if err := h.db.Where("email = ?", ownerDummy.Email).First(&existingOwner).Error; err != nil {
		h.db.Create(&ownerDummy)
		ownerID = ownerDummy.ID
	} else {
		ownerID = existingOwner.ID // Gunakan ID dari database jika user sudah ada!
	}

	// Sales Dummies
	salesDummies := []postgresRepo.UserModel{
		{
			ID:         uuid.NewString(),
			Name:       "Rina Pratiwi (Sales Dummy)",
			Email:      "rina_dummy@sikon.com",
			Password:   string(hashedPassword),
			Role:       string(domain.RoleSales),
			Phone:      "6281200000001",
			StatusText: "Online sekarang",
			IsActive:   true,
			SortOrder:  1,
		},
		{
			ID:         uuid.NewString(),
			Name:       "Dimas Aditya (Sales Dummy)",
			Email:      "dimas_dummy@sikon.com",
			Password:   string(hashedPassword),
			Role:       string(domain.RoleSales),
			Phone:      "6281200000002",
			StatusText: "Online sekarang",
			IsActive:   true,
			SortOrder:  2,
		},
		{
			ID:         uuid.NewString(),
			Name:       "Sari Lestari (Sales Dummy)",
			Email:      "sari_dummy@sikon.com",
			Password:   string(hashedPassword),
			Role:       string(domain.RoleSales),
			Phone:      "6281200000003",
			StatusText: "Balas dalam 1 jam",
			IsActive:   true,
			SortOrder:  3,
		},
		{
			ID:         uuid.NewString(),
			Name:       "Bagus Setiawan (Sales Dummy)",
			Email:      "bagus_dummy@sikon.com",
			Password:   string(hashedPassword),
			Role:       string(domain.RoleSales),
			Phone:      "6281200000004",
			StatusText: "Online sekarang",
			IsActive:   true,
			SortOrder:  4,
		},
		{
			ID:         uuid.NewString(),
			Name:       "Jonathan (Accounting Dummy)",
			Email:      "jonathan_dummy@sikon.com",
			Password:   string(hashedPassword),
			Role:       string(domain.RoleAccounting),
			Phone:      "6281200000004",
			StatusText: "Online sekarang",
			IsActive:   true,
			SortOrder:  5,
		},
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

	// --- 2. SETUP MASTER SPEC TEMPLATE (KAIN GLOBAL) ---
	specRipstopID := uuid.NewString()
	specRipstop := postgresRepo.SpecTemplateModel{
		ID:              specRipstopID,
		Name:            "Dummy Spec Ripstop Cotton",
		Spec:            "Bahan ripstop serat kotak anti robek, kuat, cocok untuk kemeja & rompi outdoor.",
		Description:     "Kain dengan konstruksi jalinan benang serat kotak presisi.",
		Composition:     "65% Cotton / 35% Polyester",
		CareInstruction: "Cuci mesin air dingin, jangan gunakan pemutih.",
		Colors: []postgresRepo.FabricColorModel{
			{ID: uuid.NewString(), SpecTemplateID: &specRipstopID, Name: "Olive", HexCode: "#4b5320"},
			{ID: uuid.NewString(), SpecTemplateID: &specRipstopID, Name: "Black", HexCode: "#000000"},
			{ID: uuid.NewString(), SpecTemplateID: &specRipstopID, Name: "Khaki", HexCode: "#c2b280"},
		},
	}

	specAmericanID := uuid.NewString()

	specAmerican := postgresRepo.SpecTemplateModel{
		ID:              specAmericanID,
		Name:            "Dummy Spec American Drill",
		Spec:            "Tekstur miring sedang, menyerap keringat dengan baik, warna tahan lama.",
		Description:     "Bahan drill populer untuk seragam lapangan dan kantor.",
		Composition:     "65% Polyester / 35% Viscose",
		CareInstruction: "Setrika suhu sedang, jemur di tempat teduh.",
		Colors: []postgresRepo.FabricColorModel{
			{ID: uuid.NewString(), SpecTemplateID: &specAmericanID, Name: "Navy", HexCode: "#1b263b"},
			{ID: uuid.NewString(), SpecTemplateID: &specAmericanID, Name: "Khaki", HexCode: "#c2b280"},
			{ID: uuid.NewString(), SpecTemplateID: &specAmericanID, Name: "Maroon", HexCode: "#800000"},
		},
	}

	specNagataID := uuid.NewString()
	specNagata := postgresRepo.SpecTemplateModel{
		ID:              specNagataID,
		Name:            "Dummy Spec Nagata Drill",
		Spec:            "Tekstur serat lebih tebal, lembut, adem dan sangat nyaman dipakai.",
		Description:     "Kain drill kelas premium untuk pakaian kerja eksklusif.",
		Composition:     "100% Cotton Premium",
		CareInstruction: "Cuci dengan warna serupa, hindari pengering panas.",
		Colors: []postgresRepo.FabricColorModel{
			{ID: uuid.NewString(), SpecTemplateID: &specNagataID, Name: "Putih", HexCode: "#ffffff"},
			{ID: uuid.NewString(), SpecTemplateID: &specNagataID, Name: "Hitam", HexCode: "#000000"},
			{ID: uuid.NewString(), SpecTemplateID: &specNagataID, Name: "Grey", HexCode: "#808080"},
		},
	}

	specLacosteID := uuid.NewString()

	specLacoste := postgresRepo.SpecTemplateModel{
		ID:              specLacosteID,
		Name:            "Dummy Spec Lacoste CVC",
		Spec:            "Rajutan pique berpori khas polo shirt, adem, dan tidak mudah berserabut.",
		Description:     "Bahan kain kaos berkerah dengan tekstur honeycomb.",
		Composition:     "60% Cotton / 40% Polyester",
		CareInstruction: "Jangan diperas terlalu kuat, gantung saat menjemur.",
		Colors: []postgresRepo.FabricColorModel{
			{ID: uuid.NewString(), SpecTemplateID: &specLacosteID, Name: "Navy", HexCode: "#1b263b"},
			{ID: uuid.NewString(), SpecTemplateID: &specLacosteID, Name: "Red", HexCode: "#ff0000"},
			{ID: uuid.NewString(), SpecTemplateID: &specLacosteID, Name: "White", HexCode: "#ffffff"},
		},
	}

	h.db.Create(&specRipstop)
	h.db.Create(&specAmerican)
	h.db.Create(&specNagata)
	h.db.Create(&specLacoste)

	// --- 2b. SETUP MASTER MATERIAL (BOM BAHAN BAKU HPP) ---
	matRipstopID := uuid.NewString()
	matRipstop := postgresRepo.MaterialModel{
		ID:        matRipstopID,
		Name:      "Dummy Mat Kain Ripstop Cotton",
		Unit:      "meter",
		UnitPrice: 25000,
		Category:  "kain",
	}

	matAmericanID := uuid.NewString()
	matAmerican := postgresRepo.MaterialModel{
		ID:        matAmericanID,
		Name:      "Dummy Mat Kain American Drill",
		Unit:      "meter",
		UnitPrice: 28000,
		Category:  "kain",
	}

	matNagataID := uuid.NewString()
	matNagata := postgresRepo.MaterialModel{
		ID:        matNagataID,
		Name:      "Dummy Mat Kain Nagata Drill",
		Unit:      "meter",
		UnitPrice: 35000,
		Category:  "kain",
	}

	matLacosteID := uuid.NewString()
	matLacoste := postgresRepo.MaterialModel{
		ID:        matLacosteID,
		Name:      "Dummy Mat Kain Lacoste CVC",
		Unit:      "meter",
		UnitPrice: 30000,
		Category:  "kain",
	}

	matKancingID := uuid.NewString()
	matKancing := postgresRepo.MaterialModel{
		ID:        matKancingID,
		Name:      "Dummy Mat Kancing Kemeja & Polo",
		Unit:      "pcs",
		UnitPrice: 1000,
		Category:  "aksesoris",
	}

	matBenangID := uuid.NewString()
	matBenang := postgresRepo.MaterialModel{
		ID:        matBenangID,
		Name:      "Dummy Mat Benang Jahit Premium",
		Unit:      "roll",
		UnitPrice: 2500,
		Category:  "aksesoris",
	}

	matResletingID := uuid.NewString()
	matResleting := postgresRepo.MaterialModel{
		ID:        matResletingID,
		Name:      "Dummy Mat Resleting Taktikal",
		Unit:      "pcs",
		UnitPrice: 5000,
		Category:  "aksesoris",
	}

	matPolybagID := uuid.NewString()
	matPolybag := postgresRepo.MaterialModel{
		ID:        matPolybagID,
		Name:      "Dummy Mat Polybag Packaging",
		Unit:      "pcs",
		UnitPrice: 500,
		Category:  "packaging",
	}

	h.db.Create(&matRipstop)
	h.db.Create(&matAmerican)
	h.db.Create(&matNagata)
	h.db.Create(&matLacoste)
	h.db.Create(&matKancing)
	h.db.Create(&matBenang)
	h.db.Create(&matResleting)
	h.db.Create(&matPolybag)

	// --- 3. SETUP 4 KATEGORI ---
	catKemeja := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Dummy Cat Kemeja"}
	catRompi := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Dummy Cat Rompi"}
	catCelana := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Dummy Cat Celana"}
	catPolo := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Dummy Cat Polo"}
	h.db.Create(&catKemeja)
	h.db.Create(&catRompi)
	h.db.Create(&catCelana)
	h.db.Create(&catPolo)

	// --- 4. UPLOAD GAMBAR DARI ASSETS KE SUPABASE STORAGE ---

	var (
		imgKemejaURL, imgRompiURL, imgCelanaURL, imgPoloURL        string
		maskFrontLong, maskBackLong, maskFrontShort, maskBackShort string
		s1FrontLong, s1BackLong, s1FrontShort, s1BackShort         string
		s2FrontLong, s2BackLong, s2FrontShort, s2BackShort         string
	)

	jobs := []struct {
		path   string
		folder string
		target *string
	}{

		// Products
		{"assets/Product/kemeja.jpg", "products", &imgKemejaURL},
		{"assets/Product/rompi.jpg", "products", &imgRompiURL},
		{"assets/Product/celana.jpg", "products", &imgCelanaURL},
		{"assets/Product/polo.jpg", "products", &imgPoloURL},

		// Mockups - Masks
		{"assets/Mockup/front-mask-long.png", "mockups", &maskFrontLong},
		{"assets/Mockup/back-mask-long.png", "mockups", &maskBackLong},
		{"assets/Mockup/front-mask-short.png", "mockups", &maskFrontShort},
		{"assets/Mockup/back-mask-short.png", "mockups", &maskBackShort},

		// Mockups - Series 1
		{"assets/Mockup/series1-front-long.png", "mockups", &s1FrontLong},
		{"assets/Mockup/series1-back-long.png", "mockups", &s1BackLong},
		{"assets/Mockup/series1-front-short.png", "mockups", &s1FrontShort},
		{"assets/Mockup/series1-back-short.png", "mockups", &s1BackShort},

		// Mockups - Series 2
		{"assets/Mockup/series2-front-long.png", "mockups", &s2FrontLong},
		{"assets/Mockup/series2-back-long.png", "mockups", &s2BackLong},
		{"assets/Mockup/series2-front-short.png", "mockups", &s2FrontShort},
		{"assets/Mockup/series2-back-short.png", "mockups", &s2BackShort},
	}

	for _, job := range jobs {
		url, err := h.uploadLocalFile(ctx, job.path, job.folder)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, fmt.Sprintf("Gagal upload seeder file [%s]: %v", job.path, err))
		}
		*job.target = url
	}

	// --- 5. SETUP PRODUK DUMMY ---
	// A. Produk Rompi
	prodRompiID := uuid.NewString()
	prodRompi := postgresRepo.ProductModel{
		ID:            prodRompiID,
		CategoryID:    catRompi.ID,
		Name:          "Dummy Prod Rompi Taktikal Lapangan",
		Description:   "Rompi lapis banyak saku untuk kebutuhan outdoor dan teknisi.",
		BasePrice:     140000,
		Slug:          "dummy-prod-rompi-taktikal-lapangan",
		GSMInfo:       "240gsm",
		FabricSummary: "Ripstop Cotton",
		Rating:        4.7,
		SoldCount:     320,
		ReviewCount:   85,
		Images: []postgresRepo.ProductImageModel{
			{ID: uuid.NewString(), ProductID: prodRompiID, ImageURL: imgRompiURL, IsPrimary: true},
		},
	}

	// B. Produk Celana
	prodCelanaID := uuid.NewString()
	prodCelana := postgresRepo.ProductModel{
		ID:            prodCelanaID,
		CategoryID:    catCelana.ID,
		Name:          "Dummy Prod Celana PDL Taktikal",
		Description:   "Celana cargo PDL dengan jahitan ganda & kantong ekstra.",
		BasePrice:     170000,
		Slug:          "dummy-prod-celana-pdl-taktikal",
		GSMInfo:       "260gsm",
		FabricSummary: "Nagata Drill",
		Rating:        4.8,
		SoldCount:     510,
		ReviewCount:   140,
		Images: []postgresRepo.ProductImageModel{
			{ID: uuid.NewString(), ProductID: prodCelanaID, ImageURL: imgCelanaURL, IsPrimary: true},
		},
	}

	// C. Produk Polo
	prodPoloID := uuid.NewString()
	prodPolo := postgresRepo.ProductModel{
		ID:            prodPoloID,
		CategoryID:    catPolo.ID,
		Name:          "Dummy Prod Kaos Polo Wangky Custom",
		Description:   "Kaos polo berkerah bahan lacoste combed adem dan elegan.",
		BasePrice:     95000,
		Slug:          "dummy-prod-kaos-polo-wangky-custom",
		GSMInfo:       "220gsm",
		FabricSummary: "Lacoste CVC",
		Rating:        4.9,
		SoldCount:     890,
		ReviewCount:   210,
		Images: []postgresRepo.ProductImageModel{
			{ID: uuid.NewString(), ProductID: prodPoloID, ImageURL: imgPoloURL, IsPrimary: true},
		},
	}

	// D. Produk Kemeja Series 1
	prodKemeja1ID := uuid.NewString()
	dm1ID := uuid.NewString()
	fab1Kemeja1 := uuid.NewString()
	prodKemeja1 := postgresRepo.ProductModel{
		ID:            prodKemeja1ID,
		CategoryID:    catKemeja.ID,
		Name:          "Dummy Prod Kemeja Series 1",
		Description:   "Kemeja taktikal Series 1 dengan template customizer canvas.",
		BasePrice:     185000,
		Slug:          "dummy-prod-kemeja-series-1",
		GSMInfo:       "210gsm",
		FabricSummary: "Ripstop Cotton",
		Rating:        4.9,
		SoldCount:     1200,
		ReviewCount:   300,
		Images: []postgresRepo.ProductImageModel{
			{ID: uuid.NewString(), ProductID: prodKemeja1ID, ImageURL: imgKemejaURL, IsPrimary: true},
		},

		Fabrics: []postgresRepo.ProductFabricModel{
			{
				ID:             fab1Kemeja1,
				ProductID:      prodKemeja1ID,
				SpecTemplateID: &specRipstop.ID,
				Name:           "Ripstop Cotton Premium",
				BasePrice:      185000,
				IsDefault:      true,
				Colors:         []postgresRepo.FabricColorModel{},
			},
		},

		DesignModel: &postgresRepo.DesignerModel{
			ID:          dm1ID,
			ProductID:   prodKemeja1ID,
			Name:        "Kemeja Series 1 — Template Canvas",
			Type:        "long_sleeve",
			Description: "Template mockup custom kemeja series 1",
			Views: []postgresRepo.ProductModelViewModel{
				{ID: uuid.NewString(), ProductModelID: dm1ID, Side: "front", ArtURL: s1FrontLong, MaskURL: maskFrontLong, Width: 1756, Height: 1920},
				{ID: uuid.NewString(), ProductModelID: dm1ID, Side: "back", ArtURL: s1BackLong, MaskURL: maskBackLong, Width: 1738, Height: 1920},
				{ID: uuid.NewString(), ProductModelID: dm1ID, Side: "front_short", ArtURL: s1FrontShort, MaskURL: maskFrontShort, Width: 1727, Height: 1920},
				{ID: uuid.NewString(), ProductModelID: dm1ID, Side: "back_short", ArtURL: s1BackShort, MaskURL: maskBackShort, Width: 1708, Height: 1920},
			},
		},
	}

	// E. Produk Kemeja Series 2
	prodKemeja2ID := uuid.NewString()
	dm2ID := uuid.NewString()
	fab1Kemeja2 := uuid.NewString()
	prodKemeja2 := postgresRepo.ProductModel{
		ID:            prodKemeja2ID,
		CategoryID:    catKemeja.ID,
		Name:          "Dummy Prod Kemeja Series 2",
		Description:   "Kemeja taktikal Series 2 dengan variasi saku & desain canvas baru.",
		BasePrice:     190000,
		Slug:          "dummy-prod-kemeja-series-2",
		GSMInfo:       "220gsm",
		FabricSummary: "American Drill",
		Rating:        4.8,
		SoldCount:     950,
		ReviewCount:   220,
		Images: []postgresRepo.ProductImageModel{
			{ID: uuid.NewString(), ProductID: prodKemeja2ID, ImageURL: imgKemejaURL, IsPrimary: true},
		},

		Fabrics: []postgresRepo.ProductFabricModel{
			{
				ID:             fab1Kemeja2,
				ProductID:      prodKemeja2ID,
				SpecTemplateID: &specAmerican.ID,
				Name:           "American Drill High",
				BasePrice:      190000,
				IsDefault:      true,
				Colors:         []postgresRepo.FabricColorModel{},
			},
		},

		DesignModel: &postgresRepo.DesignerModel{
			ID:          dm2ID,
			ProductID:   prodKemeja2ID,
			Name:        "Kemeja Series 2 — Template Canvas",
			Type:        "long_sleeve",
			Description: "Template mockup custom kemeja series 2",
			Views: []postgresRepo.ProductModelViewModel{
				{ID: uuid.NewString(), ProductModelID: dm2ID, Side: "front", ArtURL: s2FrontLong, MaskURL: maskFrontLong, Width: 1756, Height: 1920},
				{ID: uuid.NewString(), ProductModelID: dm2ID, Side: "back", ArtURL: s2BackLong, MaskURL: maskBackLong, Width: 1738, Height: 1920},
				{ID: uuid.NewString(), ProductModelID: dm2ID, Side: "front_short", ArtURL: s2FrontShort, MaskURL: maskFrontShort, Width: 1727, Height: 1920},
				{ID: uuid.NewString(), ProductModelID: dm2ID, Side: "back_short", ArtURL: s2BackShort, MaskURL: maskBackShort, Width: 1708, Height: 1920},
			},
		},
	}

	// Simpan Semua Produk Ke Database
	h.db.Create(&prodRompi)
	h.db.Create(&prodCelana)
	h.db.Create(&prodPolo)
	h.db.Create(&prodKemeja1)
	h.db.Create(&prodKemeja2)
	products := []postgresRepo.ProductModel{prodRompi, prodCelana, prodPolo, prodKemeja1, prodKemeja2}

	// Setup Resep BOM per Produk (Product Materials)
	productBOMs := []postgresRepo.ProductMaterialModel{
		// Kemeja 1 (Ripstop) -> 1.5m Ripstop (37.5k) + 7 Kancing (7k) + 1 Benang (2.5k) + 1 Polybag (0.5k) = 47.5k
		{ID: uuid.NewString(), ProductID: prodKemeja1ID, MaterialID: matRipstopID, QtyPerUnit: 1.5},
		{ID: uuid.NewString(), ProductID: prodKemeja1ID, MaterialID: matKancingID, QtyPerUnit: 7},
		{ID: uuid.NewString(), ProductID: prodKemeja1ID, MaterialID: matBenangID, QtyPerUnit: 1},
		{ID: uuid.NewString(), ProductID: prodKemeja1ID, MaterialID: matPolybagID, QtyPerUnit: 1},

		// Kemeja 2 (American Drill) -> 1.5m American (42k) + 7 Kancing (7k) + 1 Benang (2.5k) + 1 Polybag (0.5k) = 52k
		{ID: uuid.NewString(), ProductID: prodKemeja2ID, MaterialID: matAmericanID, QtyPerUnit: 1.5},
		{ID: uuid.NewString(), ProductID: prodKemeja2ID, MaterialID: matKancingID, QtyPerUnit: 7},
		{ID: uuid.NewString(), ProductID: prodKemeja2ID, MaterialID: matBenangID, QtyPerUnit: 1},
		{ID: uuid.NewString(), ProductID: prodKemeja2ID, MaterialID: matPolybagID, QtyPerUnit: 1},

		// Rompi (Ripstop) -> 1.2m Ripstop (30k) + 1 Resleting (5k) + 1 Benang (2.5k) + 1 Polybag (0.5k) = 38k
		{ID: uuid.NewString(), ProductID: prodRompiID, MaterialID: matRipstopID, QtyPerUnit: 1.2},
		{ID: uuid.NewString(), ProductID: prodRompiID, MaterialID: matResletingID, QtyPerUnit: 1},
		{ID: uuid.NewString(), ProductID: prodRompiID, MaterialID: matBenangID, QtyPerUnit: 1},
		{ID: uuid.NewString(), ProductID: prodRompiID, MaterialID: matPolybagID, QtyPerUnit: 1},

		// Celana (Nagata Drill) -> 1.5m Nagata (52.5k) + 1 Resleting (5k) + 1 Kancing (1k) + 1 Benang (2.5k) + 1 Polybag (0.5k) = 61.5k
		{ID: uuid.NewString(), ProductID: prodCelanaID, MaterialID: matNagataID, QtyPerUnit: 1.5},
		{ID: uuid.NewString(), ProductID: prodCelanaID, MaterialID: matResletingID, QtyPerUnit: 1},
		{ID: uuid.NewString(), ProductID: prodCelanaID, MaterialID: matKancingID, QtyPerUnit: 1},
		{ID: uuid.NewString(), ProductID: prodCelanaID, MaterialID: matBenangID, QtyPerUnit: 1},
		{ID: uuid.NewString(), ProductID: prodCelanaID, MaterialID: matPolybagID, QtyPerUnit: 1},

		// Polo (Lacoste CVC) -> 1.2m Lacoste (36k) + 3 Kancing (3k) + 1 Benang (2.5k) + 1 Polybag (0.5k) = 42k
		{ID: uuid.NewString(), ProductID: prodPoloID, MaterialID: matLacosteID, QtyPerUnit: 1.2},
		{ID: uuid.NewString(), ProductID: prodPoloID, MaterialID: matKancingID, QtyPerUnit: 3},
		{ID: uuid.NewString(), ProductID: prodPoloID, MaterialID: matBenangID, QtyPerUnit: 1},
		{ID: uuid.NewString(), ProductID: prodPoloID, MaterialID: matPolybagID, QtyPerUnit: 1},
	}
	for _, pb := range productBOMs {
		h.db.Create(&pb)
	}

	productBOMCost := map[string]float64{
		prodKemeja1ID: 47500,
		prodKemeja2ID: 52000,
		prodRompiID:   38000,
		prodCelanaID:  61500,
		prodPoloID:    42000,
	}

	// --- 6. SETUP CUSTOMERS ---
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

	// --- 7. SETUP EXPENSE CATEGORIES ---
	expenseCats := []postgresRepo.ExpenseCategoryModel{
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Belanja Kain & Benang", Type: string(domain.ExpenseTypeHPP)},
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Ongkos Jahit (CMT)", Type: string(domain.ExpenseTypeHPP)},
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Gaji Karyawan", Type: string(domain.ExpenseTypeOPEX)},
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Operasional Listrik", Type: string(domain.ExpenseTypeOPEX)},
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Internet & Hosting", Type: string(domain.ExpenseTypeOPEX)},
		{ID: uuid.NewString(), Name: "Dummy Cat Exp - Biaya Iklan Meta Ads", Type: string(domain.ExpenseTypeOPEX)},
	}

	for _, ec := range expenseCats {
		h.db.Create(&ec)
	}

	// --- 8. SETUP BATCH PO SEPTEMBER ---
	poSchedules := []struct {
		Name   string
		Month  int
		Year   int
		Start  string
		End    string
		Status string
	}{
		{"PO September 1", 9, 2026, "2026-08-29", "2026-09-05", string(domain.BatchPOStatusClosed)},
		{"PO September 2", 9, 2026, "2026-09-05", "2026-09-12", string(domain.BatchPOStatusClosed)},
		{"PO September 3", 9, 2026, "2026-09-12", "2026-09-19", string(domain.BatchPOStatusActive)},
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

	// --- 9. SETUP MASTER WORKER (PEKERJA KONVEKSI) ---
	workers := []postgresRepo.WorkerModel{
		{ID: uuid.NewString(), Name: "Mang Ade (Penjahit)", Phone: "081299990001", Role: string(domain.WorkerRoleTailor), SalaryType: string(domain.WorkerSalaryTypePieceRate), Status: string(domain.WorkerStatusActive)},
		{ID: uuid.NewString(), Name: "Mang Wahyu (Pemotong)", Phone: "081299990002", Role: string(domain.WorkerRoleCutter), SalaryType: string(domain.WorkerSalaryTypePieceRate), Status: string(domain.WorkerStatusActive)},
		{ID: uuid.NewString(), Name: "Kang Agus (Finishing)", Phone: "081299990003", Role: string(domain.WorkerRoleFinishing), SalaryType: string(domain.WorkerSalaryTypePieceRate), Status: string(domain.WorkerStatusActive)},
	}

	for _, w := range workers {
		h.db.Create(&w)
	}

	// --- 10. LOOPING TRANSAKSI HARIAN & WORK LOGS ---
	po1Start, _ := time.Parse("2006-01-02", "2026-08-29")
	po2Start, _ := time.Parse("2006-01-02", "2026-09-05")
	po3Start, _ := time.Parse("2006-01-02", "2026-09-12")
	po3End, _ := time.Parse("2006-01-02", "2026-09-19")
	startDate := po1Start
	endDate := po3End

	orderCounter := 1
	var po1WorkLogIDs, po2WorkLogIDs, po3WorkLogIDs []string

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		var currentPO postgresRepo.BatchPOModel
		var currentPOIndex int
		if d.Before(po2Start) {
			currentPO = batchPOs[0]
			currentPOIndex = 0
		} else if d.Before(po3Start) {
			currentPO = batchPOs[1]
			currentPOIndex = 1
		} else {
			currentPO = batchPOs[2]
			currentPOIndex = 2
		}
		poID := currentPO.ID
		activePoID := &poID

		numOrders := rand.Intn(3) + 1
		for i := 0; i < numOrders; i++ {
			cust := customers[rand.Intn(len(customers))]
			prod := products[rand.Intn(len(products))]
			qty := rand.Intn(15) + 5
			totalAmount := float64(qty) * prod.BasePrice

			var orderStatus string
			var paymentStatus string
			var isCompleted bool

			if currentPO.Status == string(domain.BatchPOStatusClosed) {
				// Aturan Closed PO:
				// Ada yang completed (Lunas 100%), sisanya ready (Partial DP 50%), TIDAK ADA yang unpaid
				if (orderCounter % 2) == 0 {
					orderStatus = string(domain.OrderStatusCompleted)
					paymentStatus = string(domain.PaymentStatusPaid)
					isCompleted = true
				} else {
					orderStatus = string(domain.OrderStatusReady)
					paymentStatus = string(domain.PaymentStatusPartial)
					isCompleted = false
				}
			} else {
				// Aturan Active PO 3:
				// Semuanya sudah DP (Partial 50%) tapi statusnya production
				orderStatus = string(domain.OrderStatusProduction)
				paymentStatus = string(domain.PaymentStatusPartial)
				isCompleted = false
			}

			createdAt := d.Add(time.Duration(rand.Intn(4)+8) * time.Hour)
			appTime := createdAt.Add(2 * time.Hour)
			approvedTimePtr := &appTime

			// Hitung Snapshot HPP Material dari Resep BOM x Qty
			unitBOMCost := productBOMCost[prod.ID]
			hppMaterialCost := float64(qty) * unitBOMCost

			orderID := uuid.NewString()
			order := postgresRepo.OrderModel{
				ID:              orderID,
				OrderNumber:     fmt.Sprintf("SEED-ORD-%04d", orderCounter),
				BatchPoID:       activePoID,
				CustomerID:      cust.ID,
				SalesID:         *cust.SalesID,
				Subtotal:        totalAmount,
				TotalAmount:     totalAmount,
				OrderStatus:     orderStatus,
				PaymentStatus:   paymentStatus,
				CreatedAt:       createdAt,
				ApprovedAt:      approvedTimePtr,
				HPPMaterialCost: hppMaterialCost,
				HPPCalculatedAt: approvedTimePtr,
			}
			h.db.Create(&order)

			// Order Item
			orderItem := postgresRepo.OrderItemModel{
				ID:         uuid.NewString(),
				OrderID:    orderID,
				ProductID:  prod.ID,
				CustomName: fmt.Sprintf("%s (%s)", prod.Name, cust.Name),
				Qty:        qty,
				Price:      prod.BasePrice,
			}
			h.db.Create(&orderItem)

			// Pembayaran Terverifikasi
			verifierID := *cust.SalesID
			payTimeDP := appTime.Add(1 * time.Hour)

			if isCompleted {
				// 1. Pembayaran DP (50%)
				dpPayment := postgresRepo.PaymentModel{
					ID:              uuid.NewString(),
					OrderID:         orderID,
					BankAccountID:   bankAccount.ID,
					Amount:          totalAmount * 0.5,
					PaymentType:     string(domain.PaymentTypeDP),
					Status:          string(domain.PaymentVerificationVerified),
					VerifiedByID:    &verifierID,
					VerifiedAt:      &payTimeDP,
					PaymentDate:     payTimeDP,
					ReferenceNumber: fmt.Sprintf("SEED-PAY-DP-%04d", orderCounter),
				}
				h.db.Create(&dpPayment)

				// 2. Pembayaran Pelunasan (50%)
				payTimeSettlement := payTimeDP.Add(48 * time.Hour)
				settlePayment := postgresRepo.PaymentModel{
					ID:              uuid.NewString(),
					OrderID:         orderID,
					BankAccountID:   bankAccount.ID,
					Amount:          totalAmount * 0.5,
					PaymentType:     string(domain.PaymentTypeSettlement),
					Status:          string(domain.PaymentVerificationVerified),
					VerifiedByID:    &verifierID,
					VerifiedAt:      &payTimeSettlement,
					PaymentDate:     payTimeSettlement,
					ReferenceNumber: fmt.Sprintf("SEED-PAY-SETTLE-%04d", orderCounter),
				}
				h.db.Create(&settlePayment)
			} else {
				// Pembayaran DP 50% (untuk Ready & Production)
				dpPayment := postgresRepo.PaymentModel{
					ID:              uuid.NewString(),
					OrderID:         orderID,
					BankAccountID:   bankAccount.ID,
					Amount:          totalAmount * 0.5,
					PaymentType:     string(domain.PaymentTypeDP),
					Status:          string(domain.PaymentVerificationVerified),
					VerifiedByID:    &verifierID,
					VerifiedAt:      &payTimeDP,
					PaymentDate:     payTimeDP,
					ReferenceNumber: fmt.Sprintf("SEED-PAY-DP-%04d", orderCounter),
				}
				h.db.Create(&dpPayment)
			}

			// Catatan Kerja Borongan (WorkLog)
			// PO Closed yang ready/completed otomatis sudah dipotong dan dijahit
			if orderStatus == string(domain.OrderStatusReady) || orderStatus == string(domain.OrderStatusCompleted) {
				// WorkLog Potong (Mang Wahyu)
				wlPotongID := uuid.NewString()
				ratePotong := 4000.0
				wlPotong := postgresRepo.WorkLogModel{
					ID:          wlPotongID,
					WorkerID:    workers[1].ID, // Mang Wahyu
					BatchPoID:   activePoID,
					OrderID:     &orderID,
					JobType:     string(domain.JobTypePotong),
					Qty:         qty,
					RatePerQty:  ratePotong,
					TotalAmount: float64(qty) * ratePotong,
					WorkDate:    d,
					Notes:       fmt.Sprintf("Potong pola kain %s (Order #%s)", prod.Name, order.OrderNumber),
					CreatedByID: ownerID,
				}
				h.db.Create(&wlPotong)

				// WorkLog Jahit (Mang Ade)
				wlJahitID := uuid.NewString()
				rateJahit := 12000.0
				wlJahit := postgresRepo.WorkLogModel{
					ID:          wlJahitID,
					WorkerID:    workers[0].ID, // Mang Ade
					BatchPoID:   activePoID,
					OrderID:     &orderID,
					JobType:     string(domain.JobTypeJahit),
					Qty:         qty,
					RatePerQty:  rateJahit,
					TotalAmount: float64(qty) * rateJahit,
					WorkDate:    d.AddDate(0, 0, 1),
					Notes:       fmt.Sprintf("Jahit rakit %s (Order #%s)", prod.Name, order.OrderNumber),
					CreatedByID: ownerID,
				}
				h.db.Create(&wlJahit)

				if currentPOIndex == 0 {
					po1WorkLogIDs = append(po1WorkLogIDs, wlPotongID, wlJahitID)
				} else {
					po2WorkLogIDs = append(po2WorkLogIDs, wlPotongID, wlJahitID)
				}

				// WorkLog Finishing (Kang Agus) jika order completed
				if orderStatus == string(domain.OrderStatusCompleted) {
					wlFinishID := uuid.NewString()
					rateFinish := 3000.0
					wlFinish := postgresRepo.WorkLogModel{
						ID:          wlFinishID,
						WorkerID:    workers[2].ID, // Kang Agus
						BatchPoID:   activePoID,
						OrderID:     &orderID,
						JobType:     string(domain.JobTypeFinishing),
						Qty:         qty,
						RatePerQty:  rateFinish,
						TotalAmount: float64(qty) * rateFinish,
						WorkDate:    d.AddDate(0, 0, 2),
						Notes:       fmt.Sprintf("Finishing & packing %s (Order #%s)", prod.Name, order.OrderNumber),
						CreatedByID: ownerID,
					}
					h.db.Create(&wlFinish)

					if currentPOIndex == 0 {
						po1WorkLogIDs = append(po1WorkLogIDs, wlFinishID)
					} else {
						po2WorkLogIDs = append(po2WorkLogIDs, wlFinishID)
					}
				}
			} else if orderStatus == string(domain.OrderStatusProduction) {
				// Untuk PO 3 yang sedang berjalan (Production):
				// Sudah dipotong, dan sebagian sudah mulai dijahit
				wlPotongID := uuid.NewString()
				ratePotong := 4000.0
				wlPotong := postgresRepo.WorkLogModel{
					ID:          wlPotongID,
					WorkerID:    workers[1].ID, // Mang Wahyu
					BatchPoID:   activePoID,
					OrderID:     &orderID,
					JobType:     string(domain.JobTypePotong),
					Qty:         qty,
					RatePerQty:  ratePotong,
					TotalAmount: float64(qty) * ratePotong,
					WorkDate:    d,
					Notes:       fmt.Sprintf("Potong pola kain %s (Order #%s)", prod.Name, order.OrderNumber),
					CreatedByID: ownerID,
				}
				h.db.Create(&wlPotong)
				po3WorkLogIDs = append(po3WorkLogIDs, wlPotongID)

				if (orderCounter % 2) == 0 {
					wlJahitID := uuid.NewString()
					rateJahit := 12000.0
					wlJahit := postgresRepo.WorkLogModel{
						ID:          wlJahitID,
						WorkerID:    workers[0].ID, // Mang Ade
						BatchPoID:   activePoID,
						OrderID:     &orderID,
						JobType:     string(domain.JobTypeJahit),
						Qty:         qty,
						RatePerQty:  rateJahit,
						TotalAmount: float64(qty) * rateJahit,
						WorkDate:    d,
						Notes:       fmt.Sprintf("Jahit proses berjalan %s (Order #%s)", prod.Name, order.OrderNumber),
						CreatedByID: ownerID,
					}
					h.db.Create(&wlJahit)
					po3WorkLogIDs = append(po3WorkLogIDs, wlJahitID)
				}
			}

			orderCounter++
		}
	}

	// --- 11. SETUP DUMMY PAYROLL (REKAP GAJI MINGGUAN) ---
	var expCMTCat postgresRepo.ExpenseCategoryModel
	for _, ec := range expenseCats {
		if ec.Name == "Dummy Cat Exp - Ongkos Jahit (CMT)" {
			expCMTCat = ec
			break
		}
	}

	expenseCounter := 0

	// 11a. Payroll PO September 1 (Closed -> Paid)
	if len(po1WorkLogIDs) > 0 {
		payrollID1 := uuid.NewString()
		p1Start, _ := time.Parse("2006-01-02", "2026-08-29")
		p1End, _ := time.Parse("2006-01-02", "2026-09-05")

		var totalAmount1 float64
		h.db.Model(&postgresRepo.WorkLogModel{}).
			Where("id IN ?", po1WorkLogIDs).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&totalAmount1)

		expID1 := uuid.NewString()
		paidAt1 := p1End.Add(17 * time.Hour)
		expense1 := postgresRepo.ExpenseModel{
			ID:                expID1,
			ExpenseCategoryID: expCMTCat.ID,
			BatchPoID:         &batchPOs[0].ID,
			Title:             "Pembayaran Rekap Gaji PAY-202609-001 (PO September 1)",
			Amount:            totalAmount1,
			ExpenseDate:       paidAt1,
			Notes:             "Otomatis dibuat dari modul Payroll PO September 1",
			CreatedByID:       ownerID,
		}
		h.db.Create(&expense1)
		expenseCounter++

		payroll1 := postgresRepo.PayrollModel{
			ID:            payrollID1,
			PayrollNumber: "PAY-202609-001",
			StartDate:     p1Start,
			EndDate:       p1End,
			TotalAmount:   totalAmount1,
			Status:        string(domain.PayrollStatusPaid),
			ExpenseID:     &expID1,
			PaidAt:        &paidAt1,
			CreatedByID:   ownerID,
		}
		h.db.Create(&payroll1)

		h.db.Model(&postgresRepo.WorkLogModel{}).
			Where("id IN ?", po1WorkLogIDs).
			Update("payroll_id", payrollID1)
	}

	// 11b. Payroll PO September 2 (Closed -> Paid)
	if len(po2WorkLogIDs) > 0 {
		payrollID2 := uuid.NewString()
		p2Start, _ := time.Parse("2006-01-02", "2026-09-05")
		p2End, _ := time.Parse("2006-01-02", "2026-09-12")

		var totalAmount2 float64
		h.db.Model(&postgresRepo.WorkLogModel{}).
			Where("id IN ?", po2WorkLogIDs).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&totalAmount2)

		expID2 := uuid.NewString()
		paidAt2 := p2End.Add(17 * time.Hour)
		expense2 := postgresRepo.ExpenseModel{
			ID:                expID2,
			ExpenseCategoryID: expCMTCat.ID,
			BatchPoID:         &batchPOs[1].ID,
			Title:             "Pembayaran Rekap Gaji PAY-202609-002 (PO September 2)",
			Amount:            totalAmount2,
			ExpenseDate:       paidAt2,
			Notes:             "Otomatis dibuat dari modul Payroll PO September 2",
			CreatedByID:       ownerID,
		}
		h.db.Create(&expense2)
		expenseCounter++

		payroll2 := postgresRepo.PayrollModel{
			ID:            payrollID2,
			PayrollNumber: "PAY-202609-002",
			StartDate:     p2Start,
			EndDate:       p2End,
			TotalAmount:   totalAmount2,
			Status:        string(domain.PayrollStatusPaid),
			ExpenseID:     &expID2,
			PaidAt:        &paidAt2,
			CreatedByID:   ownerID,
		}
		h.db.Create(&payroll2)

		h.db.Model(&postgresRepo.WorkLogModel{}).
			Where("id IN ?", po2WorkLogIDs).
			Update("payroll_id", payrollID2)
	}

	// 11c. Payroll PO September 3 (Active -> Draft)
	if len(po3WorkLogIDs) > 0 {
		payrollID3 := uuid.NewString()
		p3Start, _ := time.Parse("2006-01-02", "2026-09-12")
		p3End, _ := time.Parse("2006-01-02", "2026-09-19")

		var totalAmount3 float64
		h.db.Model(&postgresRepo.WorkLogModel{}).
			Where("id IN ?", po3WorkLogIDs).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&totalAmount3)

		payroll3 := postgresRepo.PayrollModel{
			ID:            payrollID3,
			PayrollNumber: "PAY-202609-003",
			StartDate:     p3Start,
			EndDate:       p3End,
			TotalAmount:   totalAmount3,
			Status:        string(domain.PayrollStatusDraft),
			CreatedByID:   ownerID,
		}
		h.db.Create(&payroll3)

		h.db.Model(&postgresRepo.WorkLogModel{}).
			Where("id IN ?", po3WorkLogIDs).
			Update("payroll_id", payrollID3)
	}

	// --- 12. SETUP PENGELUARAN (EXPENSES) HPP TIAP PO & OPEX ---
	var expKainCat, expListrikCat, expInternetCat, expAdsCat postgresRepo.ExpenseCategoryModel
	for _, ec := range expenseCats {
		switch ec.Name {
		case "Dummy Cat Exp - Belanja Kain & Benang":
			expKainCat = ec
		case "Dummy Cat Exp - Operasional Listrik":
			expListrikCat = ec
		case "Dummy Cat Exp - Internet & Hosting":
			expInternetCat = ec
		case "Dummy Cat Exp - Biaya Iklan Meta Ads":
			expAdsCat = ec
		}
	}

	fixedExpenses := []postgresRepo.ExpenseModel{
		// HPP Belanja Material PO September 1
		{
			ID:                uuid.NewString(),
			ExpenseCategoryID: expKainCat.ID,
			BatchPoID:         &batchPOs[0].ID,
			Title:             "Belanja Kain Ripstop & Aksesoris PO September 1",
			Amount:            4500000,
			ExpenseDate:       po1Start.Add(24 * time.Hour),
			Notes:             "Bahan baku utama untuk PO September 1",
			CreatedByID:       ownerID,
		},
		{
			ID:                uuid.NewString(),
			ExpenseCategoryID: expKainCat.ID,
			BatchPoID:         &batchPOs[0].ID,
			Title:             "Belanja Kancing & Benang PO September 1",
			Amount:            850000,
			ExpenseDate:       po1Start.Add(48 * time.Hour),
			Notes:             "Aksesoris pelengkap PO September 1",
			CreatedByID:       ownerID,
		},
		// HPP Belanja Material PO September 2
		{
			ID:                uuid.NewString(),
			ExpenseCategoryID: expKainCat.ID,
			BatchPoID:         &batchPOs[1].ID,
			Title:             "Belanja Kain American Drill & Nagata PO September 2",
			Amount:            5200000,
			ExpenseDate:       po2Start.Add(24 * time.Hour),
			Notes:             "Bahan baku utama untuk PO September 2",
			CreatedByID:       ownerID,
		},
		{
			ID:                uuid.NewString(),
			ExpenseCategoryID: expKainCat.ID,
			BatchPoID:         &batchPOs[1].ID,
			Title:             "Belanja Resleting & Polybag PO September 2",
			Amount:            950000,
			ExpenseDate:       po2Start.Add(48 * time.Hour),
			Notes:             "Aksesoris pelengkap PO September 2",
			CreatedByID:       ownerID,
		},
		// HPP Belanja Material PO September 3
		{
			ID:                uuid.NewString(),
			ExpenseCategoryID: expKainCat.ID,
			BatchPoID:         &batchPOs[2].ID,
			Title:             "Belanja Kain Lacoste CVC & Ripstop PO September 3",
			Amount:            4800000,
			ExpenseDate:       po3Start.Add(24 * time.Hour),
			Notes:             "Bahan baku utama untuk PO September 3",
			CreatedByID:       ownerID,
		},
		{
			ID:                uuid.NewString(),
			ExpenseCategoryID: expKainCat.ID,
			BatchPoID:         &batchPOs[2].ID,
			Title:             "Belanja Benang & Kancing PO September 3",
			Amount:            750000,
			ExpenseDate:       po3Start.Add(48 * time.Hour),
			Notes:             "Aksesoris pelengkap PO September 3",
			CreatedByID:       ownerID,
		},

		// OPEX Bulanan
		{
			ID:                uuid.NewString(),
			ExpenseCategoryID: expListrikCat.ID,
			BatchPoID:         nil,
			Title:             "Tagihan Listrik Workshop Mesin Jahit Agustus/September",
			Amount:            850000,
			ExpenseDate:       po2Start,
			Notes:             "Listrik operasional workshop bulanan",
			CreatedByID:       ownerID,
		},
		{
			ID:                uuid.NewString(),
			ExpenseCategoryID: expInternetCat.ID,
			BatchPoID:         nil,
			Title:             "Langganan Internet Kantor & Hosting Server",
			Amount:            650000,
			ExpenseDate:       po2Start.Add(72 * time.Hour),
			Notes:             "Internet & server SIKOn",
			CreatedByID:       ownerID,
		},
		{
			ID:                uuid.NewString(),
			ExpenseCategoryID: expAdsCat.ID,
			BatchPoID:         nil,
			Title:             "Biaya Campaign Iklan Meta Ads (FB & IG)",
			Amount:            1500000,
			ExpenseDate:       po3Start,
			Notes:             "Promosi kemeja & rompi online",
			CreatedByID:       ownerID,
		},
	}

	for _, fe := range fixedExpenses {
		h.db.Create(&fe)
		expenseCounter++
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menyuntikkan data Seeder lengkap dengan Upload Gambar Produk, Spec Templates Kain, Master Material & BOM HPP, Canvas Mockup, & Master Workers/Payroll!", fiber.Map{
		"total_orders_generated":    orderCounter - 1,
		"total_expenses_generated":  expenseCounter,
		"total_workers_generated":   len(workers),
		"total_work_logs_generated": len(po1WorkLogIDs) + len(po2WorkLogIDs) + len(po3WorkLogIDs),
		"total_materials_generated": 8,
		"total_boms_generated":      len(productBOMs),
	})
}
