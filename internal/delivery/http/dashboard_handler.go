package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type DashboardHandler interface {
	GetSummary(c *fiber.Ctx) error
	GetSalesReport(c *fiber.Ctx) error
	GetReceivablesReport(c *fiber.Ctx) error
	GetOverview(c *fiber.Ctx) error
}

type dashboardHandler struct {
	dashboardUsecase domain.DashboardUsecase
}

func NewDashboardHandler(du domain.DashboardUsecase) DashboardHandler {
	return &dashboardHandler{
		dashboardUsecase: du,
	}
}

// @Summary Get Dashboard Summary
// @Description Mengambil metrik utama untuk halaman dashboard utama. Menghitung total omzet, uang masuk, sisa tagihan/piutang (akumulatif all-time), dan status pesanan.
// @Tags Dashboard
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Filter Tanggal Mulai (Format: YYYY-MM-DD)"
// @Param end_date query string false "Filter Tanggal Akhir (Format: YYYY-MM-DD)"
// @Success 200 {object} utils.SuccessResponse[dto.DashboardSummaryResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Router /dashboard/summary [get]
func (h *dashboardHandler) GetSummary(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	userRole, _ := c.Locals("userRole").(domain.Role)

	salesIDFilter := ""
	// 🚨 ENFORCE DASHBOARD SCOPING:
	// Jika yang login adalah Sales, paksakan SalesID ke ID dirinya
	if userRole == domain.RoleSales {
		salesIDFilter = userID
	}

	filter := domain.DashboardFilter{
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		SalesID:   salesIDFilter, // Kunci SalesID
	}

	summary, err := h.dashboardUsecase.GetSummary(c.Context(), filter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data metrik dashboard", dto.ToDashboardSummaryResponse(summary))
}

// @Summary Get Sales Report (Grafik)
// @Description Mengambil laporan penjualan harian untuk kebutuhan grafik frontend.
// @Tags Dashboard
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Filter Tanggal Mulai (Format: YYYY-MM-DD)"
// @Param end_date query string false "Filter Tanggal Akhir (Format: YYYY-MM-DD)"
// @Success 200 {object} utils.SuccessResponse[[]dto.SalesReportItemResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Router /dashboard/sales-report [get]
func (h *dashboardHandler) GetSalesReport(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	userRole, _ := c.Locals("userRole").(domain.Role)

	salesIDFilter := ""
	if userRole == domain.RoleSales {
		salesIDFilter = userID
	}

	filter := domain.DashboardFilter{
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		SalesID:   salesIDFilter, // Kunci SalesID untuk grafik
	}

	report, err := h.dashboardUsecase.GetSalesReport(c.Context(), filter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data laporan penjualan", dto.ToSalesReportItemResponseList(report))
}

// Pastikan menambahkan GetReceivablesReport(c *fiber.Ctx) error di interface DashboardHandler

// @Summary Get Receivables Report (Laporan Piutang)
// @Description Mengambil daftar pesanan berjalan (Pending/Production) yang belum lunas untuk kebutuhan tabel cetak PDF.
// @Tags Dashboard
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.SuccessResponse[[]dto.ReceivableReportItemResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Router /dashboard/receivables-report [get]
func (h *dashboardHandler) GetReceivablesReport(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	userRole, _ := c.Locals("userRole").(domain.Role)

	salesIDFilter := ""
	if userRole == domain.RoleSales {
		salesIDFilter = userID
	}

	// Oper salesIDFilter ke Usecase
	report, err := h.dashboardUsecase.GetReceivablesReport(c.Context(), salesIDFilter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil laporan piutang pesanan", dto.ToReceivableReportItemResponseList(report))
}

// @Summary Get Dashboard Overview (Single Aggregated Endpoint)
// @Description Mengambil seluruh ringkasan data dashboard dalam 1 request HTTP (Summary, Active Batch PO, Action Required Badges, Recent Orders, & Recent Payments).
// @Tags Dashboard
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.SuccessResponse[dto.DashboardOverviewResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Router /dashboard/overview [get]
func (h *dashboardHandler) GetOverview(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	userRole, _ := c.Locals("userRole").(domain.Role)

	salesIDFilter := ""
	if userRole == domain.RoleSales {
		salesIDFilter = userID
	}

	overview, err := h.dashboardUsecase.GetOverview(c.Context(), salesIDFilter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data ringkasan dashboard", dto.ToDashboardOverviewResponse(overview))
}
