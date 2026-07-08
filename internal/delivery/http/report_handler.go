package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type ReportHandler interface {
	GetDailyReport(c *fiber.Ctx) error
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
