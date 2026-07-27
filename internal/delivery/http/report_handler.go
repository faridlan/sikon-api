package http

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type ReportHandler interface {
	GetDailyReport(c *fiber.Ctx) error
	GenerateDailyReport(c *fiber.Ctx) error
	GetPOSummaryReport(c *fiber.Ctx) error
	GetReceivablesReport(c *fiber.Ctx) error
	GetMonthlyReport(c *fiber.Ctx) error
}

type reportHandler struct {
	reportUsecase domain.ReportUsecase
}

func NewReportHandler(ru domain.ReportUsecase) ReportHandler {
	return &reportHandler{reportUsecase: ru}
}

// @Summary Get Daily Report
// @Description Mengambil snapshot harian, status PO aktif, dan piutang tertunggak untuk laporan harian.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param date query string false "Tanggal laporan (Format: YYYY-MM-DD)"
// @Success 200 {object} utils.SuccessResponse[dto.ReportResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /reports/daily [get]
func (h *reportHandler) GetDailyReport(c *fiber.Ctx) error {
	date := c.Query("date")
	report, err := h.reportUsecase.GetDailyReport(c.Context(), date)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data daily report", dto.ToReportResponse(report))
}

// @Summary Generate Daily Report (New)
// @Description Mengambil laporan harian konveksi yang terstruktur. Mencakup informasi kuota PO yang aktif, ringkasan order, kalkulasi finansial, dan detail QTY per kategori produk untuk masing-masing sales.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param date query string false "Tanggal laporan (Format: YYYY-MM-DD). Jika kosong, akan menggunakan tanggal hari ini."
// @Success 200 {object} utils.SuccessResponse[dto.DailyReportResponse]
// @Failure 400,500 {object} utils.ErrorResponse
// @Router /reports/daily/generate [get]
func (h *reportHandler) GenerateDailyReport(c *fiber.Ctx) error {
	date := c.Query("date")

	// Memanggil method Usecase yang baru
	report, err := h.reportUsecase.GenerateDailyReport(c.Context(), date)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data laporan harian", dto.ToDailyReportResponse(report)) // Pastikan menggunakan DTO yang baru
}

// @Summary Get PO Close / Summary Report
// @Description Mengambil rekapitulasi data PO tertentu, termasuk total produk yang harus diproduksi dan tagihan finansial.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param po_id path string true "ID dari Batch PO"
// @Success 200 {object} utils.SuccessResponse[dto.POSummaryResponse]
// @Failure 400,404,500 {object} utils.ErrorResponse
// @Router /reports/po/{po_id}/summary [get]
func (h *reportHandler) GetPOSummaryReport(c *fiber.Ctx) error {
	poID := c.Params("po_id")

	report, err := h.reportUsecase.GetPOSummaryReport(c.Context(), poID)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil rekap laporan PO", dto.ToPOSummaryResponse(report))
}

// @Summary Get Detailed Receivables Report
// @Description Menampilkan daftar detail pelanggan dan sales yang masih memiliki piutang (Outstanding Amount > 0).
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.SuccessResponse[[]dto.ReceivableDetailResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /reports/receivables [get]
func (h *reportHandler) GetReceivablesReport(c *fiber.Ctx) error {
	data, err := h.reportUsecase.GetReceivablesDetailReport(c.Context())
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(
		c,
		fiber.StatusOK,
		"Successfully fetched detailed receivables report",
		dto.ToReceivableDetailListResponse(data),
	)
}

// @Summary Get Monthly Dashboard Report
// @Description Mengambil laporan performa penjualan, omset, arus kas, dan kinerja sales per bulan.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param month query int false "Bulan (1-12) - Default: Bulan saat ini"
// @Param year query int false "Tahun - Default: Tahun saat ini"
// @Success 200 {object} utils.SuccessResponse[dto.MonthlyReportResponse]
// @Router /reports/monthly [get]
func (h *reportHandler) GetMonthlyReport(c *fiber.Ctx) error {
	// Default ke bulan & tahun sekarang jika query tidak diisi
	now := time.Now()
	month := c.QueryInt("month", int(now.Month()))
	year := c.QueryInt("year", now.Year())

	data, err := h.reportUsecase.GetMonthlyReport(c.Context(), month, year)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(
		c,
		fiber.StatusOK,
		"Successfully fetched monthly report",
		dto.ToMonthlyReportResponse(data),
	)
}
