package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SlogMiddleware menggantikan logger bawaan Fiber menjadi JSON Structured Logging
func SlogMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Lanjutkan eksekusi request ke handler berikutnya
		err := c.Next()

		// Kalkulasi durasi
		latency := time.Since(start)

		// Ambil status code (bisa jadi dari error yang dilempar handler)
		status := c.Response().StatusCode()
		if err != nil {
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		// Ambil Request ID yang digenerate oleh Fiber (jika ada)
		requestID := c.GetRespHeader(fiber.HeaderXRequestID)

		// Siapkan atribut log
		attrs := []any{
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.String("latency", latency.String()),
			slog.String("ip", c.IP()),
			slog.String("user_agent", c.Get(fiber.HeaderUserAgent)),
		}

		// Bedakan level log berdasarkan status code HTTP
		switch {
		case status >= 500:
			slog.Error("Server Error Request", attrs...)
		case status >= 400:
			slog.Warn("Client Error Request", attrs...)
		default:
			slog.Info("HTTP Request", attrs...)
		}

		return err
	}
}
