package http

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type ReportHandler interface {
	GetAccountingReport(c *fiber.Ctx) error
	GetProductionReport(c *fiber.Ctx) error
	GetPOSummaryReport(c *fiber.Ctx) error
	GetReceivablesReport(c *fiber.Ctx) error
	GetDailyReport(c *fiber.Ctx) error // BARU: Endpoint Laporan Harian
}

type reportHandler struct {
	reportUsecase domain.ReportUsecase
}

func NewReportHandler(ru domain.ReportUsecase) ReportHandler {
	return &reportHandler{reportUsecase: ru}
}

// @Summary Get Accounting Report
// @Description Mengambil laporan keuangan (Omset, Kas Masuk, Piutang, Pengeluaran, Laba Bersih, dan Arus Kas) berdasarkan rentang tanggal.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Tanggal Awal (Format: YYYY-MM-DD)"
// @Param end_date query string false "Tanggal Akhir (Format: YYYY-MM-DD)"
// @Success 200 {object} utils.SuccessResponse[dto.AccountingReportResponse]
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Router /reports/accounting [get]
func (h *reportHandler) GetAccountingReport(c *fiber.Ctx) error {
	now := time.Now()

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if startDateStr != "" {
		parsedStart, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, "Format start_date tidak valid (Gunakan YYYY-MM-DD)")
		}
		startDate = parsedStart
	}

	endDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	if endDateStr != "" {
		parsedEnd, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, "Format end_date tidak valid (Gunakan YYYY-MM-DD)")
		}
		endDate = time.Date(parsedEnd.Year(), parsedEnd.Month(), parsedEnd.Day(), 23, 59, 59, 0, parsedEnd.Location())
	}

	report, err := h.reportUsecase.GetAccountingReport(c.Context(), startDate, endDate)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil laporan akuntansi", dto.ToAccountingReportResponse(report))
}

// @Summary Get Production Report
// @Description Mengambil laporan produksi, tagihan, HPP (Pengeluaran), Laba Produksi, dan performa sales berdasarkan Edisi PO (Bulan & Tahun).
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param month query int false "Bulan Target PO (1-12) - Default: Bulan saat ini"
// @Param year query int false "Tahun Target PO - Default: Tahun saat ini"
// @Success 200 {object} utils.SuccessResponse[dto.ProductionReportResponse]
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Router /reports/production [get]
func (h *reportHandler) GetProductionReport(c *fiber.Ctx) error {
	now := time.Now()

	month := c.QueryInt("month", int(now.Month()))
	year := c.QueryInt("year", now.Year())

	report, err := h.reportUsecase.GetProductionReport(c.Context(), month, year)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil laporan produksi", dto.ToProductionReportResponse(report))
}

// @Summary Get PO Close / Summary Report
// @Description Mengambil rekapitulasi data PO tertentu, termasuk total produk yang harus diproduksi, tagihan finansial, HPP, Net Profit, dan spesifik piutang customer.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param po_id path string true "ID dari Batch PO"
// @Success 200 {object} utils.SuccessResponse[dto.POSummaryResponse]
// @Failure 400,404,500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
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
// @Description Menampilkan daftar detail pelanggan dan sales yang masih memiliki piutang dengan opsi filter PO dan sorting.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param batch_po_id query string false "Filter berdasarkan ID Batch PO (UUID)"
// @Param order_status query string false "Filter berdasarkan status order (production, completed)"
// @Param sort_by query string false "Urutkan data (amount_desc, amount_asc, date_asc, date_desc)"
// @Success 200 {object} utils.SuccessResponse[[]dto.ReceivableDetailResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Router /reports/receivables [get]
func (h *reportHandler) GetReceivablesReport(c *fiber.Ctx) error {
	filter := domain.ReceivablesFilter{
		BatchPoID:   c.Query("batch_po_id"),
		OrderStatus: c.Query("order_status"),
		SortBy:      c.Query("sort_by"),
	}

	data, err := h.reportUsecase.GetReceivablesDetailReport(c.Context(), filter)
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

// @Summary Get Daily Tactical Report
// @Description Mengambil laporan harian operasional (briefing pagi) berdasarkan PO yang sedang aktif.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param date query string false "Tanggal Laporan (Format: YYYY-MM-DD) - Default: Hari ini"
// @Success 200 {object} utils.SuccessResponse[dto.DailyReportResponse]
// @Failure 400,500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Router /reports/daily [get]
func (h *reportHandler) GetDailyReport(c *fiber.Ctx) error {
	dateStr := c.Query("date")
	targetDate := time.Now()

	if dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, "Format date tidak valid (Gunakan YYYY-MM-DD)")
		}
		targetDate = parsed
	}

	report, err := h.reportUsecase.GetDailyReport(c.Context(), targetDate)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data laporan harian", dto.ToDailyReportResponse(report))
}
