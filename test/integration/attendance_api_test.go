package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

func attendanceRequest(t *testing.T, method, path string, payload any) *http.Request {
	t.Helper()
	body, err := json.Marshal(payload)
	assert.NoError(t, err)
	req := tests.AuthenticatedRequest(method, path, bytes.NewReader(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestAttendanceCreateAndBatch_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	worker1 := postgres.WorkerModel{ID: uuid.NewString(), Name: "Worker Attendance Satu", Role: "tailor", SalaryType: "daily", DailyRate: 100000, Status: "active"}
	worker2 := postgres.WorkerModel{ID: uuid.NewString(), Name: "Worker Attendance Dua", Role: "cutter", SalaryType: "daily", DailyRate: 80000, Status: "active"}
	assert.NoError(t, db.Create(&worker1).Error)
	assert.NoError(t, db.Create(&worker2).Error)

	t.Run("success - record attendance", func(t *testing.T) {
		resp, err := app.Test(attendanceRequest(t, "POST", "/api/attendances", dto.AttendanceCreateRequest{
			WorkerID: worker1.ID, AttendanceDate: "2026-08-28", Status: "present", Notes: "Tepat waktu",
		}), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var result utils.SuccessResponse[dto.AttendanceResponse]
		body, _ := io.ReadAll(resp.Body)
		assert.NoError(t, json.Unmarshal(body, &result))
		assert.Equal(t, worker1.ID, result.Data.WorkerID)
		assert.Equal(t, "present", result.Data.Status)
		assert.Equal(t, float64(100000), result.Data.TotalAmount)
		assert.NotEmpty(t, result.Data.ID)
	})

	t.Run("failure - duplicate and invalid payload", func(t *testing.T) {
		resp, err := app.Test(attendanceRequest(t, "POST", "/api/attendances", dto.AttendanceCreateRequest{
			WorkerID: worker1.ID, AttendanceDate: "2026-08-28", Status: "present",
		}), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

		resp, err = app.Test(attendanceRequest(t, "POST", "/api/attendances", dto.AttendanceCreateRequest{
			WorkerID: worker1.ID, AttendanceDate: "bad-date", Status: "invalid",
		}), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("success - record batch attendance", func(t *testing.T) {
		resp, err := app.Test(attendanceRequest(t, "POST", "/api/attendances/batch", dto.BatchAttendanceRequest{
			AttendanceDate: "2026-08-29",
			Items: []dto.BatchAttendanceItemRequest{
				{WorkerID: worker1.ID, Status: "half_day"},
				{WorkerID: worker2.ID, Status: "present"},
			},
		}), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var result utils.SuccessResponse[[]dto.AttendanceResponse]
		body, _ := io.ReadAll(resp.Body)
		assert.NoError(t, json.Unmarshal(body, &result))
		assert.Len(t, result.Data, 2)
		assert.Equal(t, float64(50000), result.Data[0].TotalAmount)
	})

	t.Run("failure - empty batch", func(t *testing.T) {
		resp, err := app.Test(attendanceRequest(t, "POST", "/api/attendances/batch", dto.BatchAttendanceRequest{
			AttendanceDate: "2026-08-30", Items: nil,
		}), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestAttendanceReadUpdateDelete_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)
	worker := postgres.WorkerModel{ID: uuid.NewString(), Name: "Worker Attendance CRUD", Role: "tailor", SalaryType: "daily", DailyRate: 120000, Status: "active"}
	assert.NoError(t, db.Create(&worker).Error)
	attendance := postgres.AttendanceModel{
		ID: uuid.NewString(), WorkerID: worker.ID, AttendanceDate: time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC),
		Status: "present", WorkDurationIndex: 1, DailyRate: 120000, TotalAmount: 120000,
	}
	// Use the same normalized test user ID as SetupTestApp's default user.
	attendance.CreatedByID = uuid.NewMD5(uuid.NameSpaceOID, []byte("test-user")).String()
	assert.NoError(t, db.Create(&attendance).Error)

	t.Run("success - get and list", func(t *testing.T) {
		resp, err := app.Test(tests.AuthenticatedRequest("GET", "/api/attendances/"+attendance.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		var detail utils.SuccessResponse[dto.AttendanceResponse]
		body, _ := io.ReadAll(resp.Body)
		assert.NoError(t, json.Unmarshal(body, &detail))
		assert.Equal(t, attendance.ID, detail.Data.ID)
		assert.NotNil(t, detail.Data.Worker)

		resp, err = app.Test(tests.AuthenticatedRequest("GET", "/api/attendances?worker_id="+worker.ID+"&status=present", nil, "test-user", "test-user@sikon.com", domain.RoleOwner), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		var list struct {
			Data []dto.AttendanceResponse `json:"data"`
		}
		body, _ = io.ReadAll(resp.Body)
		assert.NoError(t, json.Unmarshal(body, &list))
		assert.Len(t, list.Data, 1)
	})

	t.Run("failure - invalid id and not found", func(t *testing.T) {
		resp, err := app.Test(tests.AuthenticatedRequest("GET", "/api/attendances/not-a-uuid", nil, "test-user", "test-user@sikon.com", domain.RoleOwner), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		resp, err = app.Test(tests.AuthenticatedRequest("GET", "/api/attendances/"+uuid.NewString(), nil, "test-user", "test-user@sikon.com", domain.RoleOwner), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("success - update and delete", func(t *testing.T) {
		index := 0.5
		resp, err := app.Test(attendanceRequest(t, "PUT", "/api/attendances/"+attendance.ID, dto.AttendanceUpdateRequest{
			Status: "half_day", WorkDurationIndex: &index, Notes: "Pulang lebih awal",
		}), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		var updated utils.SuccessResponse[dto.AttendanceResponse]
		body, _ := io.ReadAll(resp.Body)
		assert.NoError(t, json.Unmarshal(body, &updated))
		assert.Equal(t, float64(60000), updated.Data.TotalAmount)

		resp, err = app.Test(tests.AuthenticatedRequest("DELETE", "/api/attendances/"+attendance.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("failure - invalid update and unauthenticated", func(t *testing.T) {
		resp, err := app.Test(attendanceRequest(t, "PUT", "/api/attendances/not-a-uuid", dto.AttendanceUpdateRequest{Status: "present"}), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		resp, err = app.Test(httptest.NewRequest("GET", "/api/attendances", nil), -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}
