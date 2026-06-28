package utils

import (
	"errors"
	"log/slog"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/gofiber/fiber/v2"
)

type ErrorResponse struct {
	Error  string `json:"error" example:"pesan error"`
	Detail string `json:"detail,omitempty"`
}

type PaginatedResponse[T any] struct {
	Message string                `json:"message"`
	Data    []T                   `json:"data"`
	Meta    domain.PaginationMeta `json:"meta"`
}

type SuccessResponse[T any] struct {
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

type EmptyObj struct{}

func SendError(c *fiber.Ctx, statusCode int, message string, detail ...string) error {
	resp := ErrorResponse{
		Error: message,
	}

	errDetail := ""
	if len(detail) > 0 && detail[0] != "" {
		errDetail = detail[0]
	}

	// 🚨 AMBIL TRACE ID
	requestID := c.GetRespHeader(fiber.HeaderXRequestID)

	// Pisahkan pencatatan log berdasarkan tingkat keparahan
	if statusCode >= fiber.StatusInternalServerError {
		// Log 500 sebagai CRITICAL ERROR
		slog.Error("CRITICAL SERVER ERROR",
			slog.String("request_id", requestID),
			slog.Int("status_code", statusCode),
			slog.String("path", c.Path()),
			slog.String("method", c.Method()),
			slog.String("error_message", message),
			slog.String("sql_detail", errDetail),
		)
		resp.Detail = "" // Sembunyikan detail SQL/Server dari End User (Keamanan)
	} else if statusCode >= fiber.StatusBadRequest {
		// Log 400-499 sebagai WARN (misal: validasi gagal, PO ditutup)
		slog.Warn("Bad Request / Forbidden",
			slog.String("request_id", requestID),
			slog.Int("status_code", statusCode),
			slog.String("path", c.Path()),
			slog.String("error_message", message),
			slog.String("detail", errDetail),
		)
		resp.Detail = errDetail // Boleh ditampilkan ke End User
	}

	return c.Status(statusCode).JSON(resp)
}

func HandleDomainError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return SendError(c, fiber.StatusNotFound, err.Error())

	case errors.Is(err, domain.ErrConflict):
		return SendError(c, fiber.StatusConflict, err.Error())

	case errors.Is(err, domain.ErrBadParamInput):
		return SendError(c, fiber.StatusBadRequest, err.Error())

	// 🚨 TAMBAHKAN BARIS INI UNTUK MENANGKAP ERROR PO TUTUP
	case errors.Is(err, domain.ErrForbidden):
		return SendError(c, fiber.StatusForbidden, err.Error())

	case errors.Is(err, domain.ErrLimitExceeded):
		return SendError(c, fiber.StatusTooManyRequests, err.Error())

	default:
		return SendError(c, fiber.StatusInternalServerError, domain.ErrInternalServerError.Error(), err.Error())
	}
}

func SendSuccessPaginated(c *fiber.Ctx, message string, data any, meta domain.PaginationMeta) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": message,
		"data":    data,
		"meta":    meta,
	})
}

func SendSuccess(c *fiber.Ctx, statusCode int, message string, data any) error {
	return c.Status(statusCode).JSON(fiber.Map{
		"message": message,
		"data":    data,
	})
}
