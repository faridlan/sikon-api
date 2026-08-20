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
	uniqueFileName := fmt.Sprintf("%s_%s", fileNameOnly, uuid.NewString()[:8])
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

	// Hapus Relasi Produk
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

	// --- 8. SETUP BATCH PO ---
	poSchedules := []struct {
		Name   string
		Month  int
		Year   int
		Start  string
		End    string
		Status string
	}{
		{"PO 1 AGUSTUS 2026", 8, 2026, "2026-08-01", "2026-08-07", string(domain.BatchPOStatusClosed)},
		{"PO 2 AGUSTUS 2026", 8, 2026, "2026-08-08", "2026-08-15", string(domain.BatchPOStatusActive)},
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
		{ID: uuid.NewString(), Name: "Kang Agus (Finishing)", Phone: "081299990003", Role: string(domain.WorkerRoleFinishing), SalaryType: string(domain.WorkerSalaryTypeDaily), Status: string(domain.WorkerStatusActive)},
	}

	for _, w := range workers {
		h.db.Create(&w)
	}

	// --- 10. LOOPING TRANSAKSI HARIAN & WORK LOGS ---
	startDate, _ := time.Parse("2006-01-02", "2026-08-01")
	endDate, _ := time.Parse("2006-01-02", "2026-08-12")
	orderCounter := 1
	expenseCounter := 0
	var generatedWorkLogIDs []string

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
			statusOptions := []string{
				string(domain.OrderStatusQuotation),
				string(domain.OrderStatusProduction),
				string(domain.OrderStatusReady),
				string(domain.OrderStatusCompleted),
			}

			orderStatus := statusOptions[rand.Intn(len(statusOptions))]
			var approvedTimePtr *time.Time
			createdAt := d.Add(8 * time.Hour)
			if orderStatus == string(domain.OrderStatusQuotation) {
				paymentStatus = string(domain.PaymentStatusUnpaid)
				paidAmount = 0
				approvedTimePtr = nil
			} else {
				appTime := d.Add(10 * time.Hour)
				approvedTimePtr = &appTime
			}

			orderID := uuid.NewString()

			order := postgresRepo.OrderModel{
				ID:            orderID,
				OrderNumber:   fmt.Sprintf("SEED-ORD-%04d", orderCounter),
				BatchPoID:     activePoID,
				CustomerID:    cust.ID,
				SalesID:       *cust.SalesID,
				Subtotal:      totalAmount,
				TotalAmount:   totalAmount,
				OrderStatus:   orderStatus,
				PaymentStatus: paymentStatus,
				CreatedAt:     createdAt,
				ApprovedAt:    approvedTimePtr,
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

			if paidAmount > 0 && approvedTimePtr != nil {
				payTime := approvedTimePtr.Add(2 * time.Hour)
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

		// Generate Catatan Kerja Borongan (WorkLog) Harian
		tailorWorker := workers[0]
		workLogQty := rand.Intn(20) + 5
		ratePerQty := 12000.0
		workLogID := uuid.NewString()
		workLog := postgresRepo.WorkLogModel{
			ID:          workLogID,
			WorkerID:    tailorWorker.ID,
			BatchPoID:   activePoID,
			JobType:     string(domain.JobTypeJahit),
			Qty:         workLogQty,
			RatePerQty:  ratePerQty,
			TotalAmount: float64(workLogQty) * ratePerQty,
			WorkDate:    d,
			Notes:       "Setor jahitan kemeja taktikal",
			CreatedByID: ownerID, // 👈 Gunakan ownerID di sini!
		}
		h.db.Create(&workLog)
		generatedWorkLogIDs = append(generatedWorkLogIDs, workLogID)

		if rand.Intn(100) < 60 {
			expCat := expenseCats[rand.Intn(len(expenseCats))]
			var poIDPtr *string
			var expAmount float64
			if expCat.Type == string(domain.ExpenseTypeHPP) {
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

	// --- 11. SETUP DUMMY PAYROLL (REKAP GAJI MINGGUAN) ---
	if len(generatedWorkLogIDs) > 0 {
		payrollID := uuid.NewString()
		payrollStart, _ := time.Parse("2006-01-02", "2026-08-01")
		payrollEnd, _ := time.Parse("2006-01-02", "2026-08-07")

		var totalPayrollAmount float64
		h.db.Model(&postgresRepo.WorkLogModel{}).
			Where("id IN ?", generatedWorkLogIDs[:len(generatedWorkLogIDs)/2]).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&totalPayrollAmount)

		payroll := postgresRepo.PayrollModel{
			ID:            payrollID,
			PayrollNumber: "PAY-202608-001",
			StartDate:     payrollStart,
			EndDate:       payrollEnd,
			TotalAmount:   totalPayrollAmount,
			Status:        string(domain.PayrollStatusDraft),
			CreatedByID:   ownerID, // 👈 Gunakan ownerID di sini!
		}
		h.db.Create(&payroll)

		// Linking WorkLogs ke Payroll
		h.db.Model(&postgresRepo.WorkLogModel{}).
			Where("id IN ?", generatedWorkLogIDs[:len(generatedWorkLogIDs)/2]).
			Update("payroll_id", payrollID)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menyuntikkan data Seeder lengkap dengan Upload Gambar Produk, Spec Templates Kain, Canvas Mockup, & Master Workers/Payroll!", fiber.Map{
		"total_orders_generated":    orderCounter - 1,
		"total_expenses_generated":  expenseCounter,
		"total_workers_generated":   len(workers),
		"total_work_logs_generated": len(generatedWorkLogIDs),
	})
}
