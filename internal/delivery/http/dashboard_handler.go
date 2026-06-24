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
// @Router /dashboard/summary [get]
func (h *dashboardHandler) GetSummary(c *fiber.Ctx) error {
	// Tangkap query param dari URL (otomatis kosong jika tidak dikirim)
	filter := domain.DashboardFilter{
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
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
// @Router /dashboard/sales-report [get]
func (h *dashboardHandler) GetSalesReport(c *fiber.Ctx) error {
	filter := domain.DashboardFilter{
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
	}

	report, err := h.dashboardUsecase.GetSalesReport(c.Context(), filter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data laporan penjualan", dto.ToSalesReportItemResponseList(report))
}
