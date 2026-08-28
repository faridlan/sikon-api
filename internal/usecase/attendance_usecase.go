package usecase

import (
	"context"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type attendanceUsecase struct {
	attendanceRepo domain.AttendanceRepository
	workerRepo     domain.WorkerRepository
	contextTimeout time.Duration
}

func NewAttendanceUsecase(repo domain.AttendanceRepository, workerRepo domain.WorkerRepository, timeout time.Duration) domain.AttendanceUsecase {
	return &attendanceUsecase{
		attendanceRepo: repo,
		workerRepo:     workerRepo,
		contextTimeout: timeout,
	}
}

// Helper kalkulasi WorkDurationIndex berdasarkan status
func calculateDurationIndex(status domain.AttendanceStatus, manualIndex *float64) float64 {
	if manualIndex != nil && *manualIndex >= 0 {
		return *manualIndex
	}

	switch status {
	case domain.AttendanceStatusPresent:
		return 1.00
	case domain.AttendanceStatusHalfDay:
		return 0.50
	case domain.AttendanceStatusPermission, domain.AttendanceStatusAlpha:
		return 0.00
	default:
		return 1.00
	}
}

func (u *attendanceUsecase) RecordAttendance(c context.Context, input domain.AttendanceCreateInput) (*domain.Attendance, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.WorkerID == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Worker ID harus diisi")
	}
	if input.AttendanceDate.IsZero() {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tanggal absensi tidak valid")
	}

	// Normalisasi tanggal YYYY-MM-DD
	cleanDate := time.Date(
		input.AttendanceDate.Year(),
		input.AttendanceDate.Month(),
		input.AttendanceDate.Day(),
		0, 0, 0, 0, time.UTC,
	)

	// 1. Ambil data Worker
	worker, err := u.workerRepo.GetByID(ctx, input.WorkerID)
	if err != nil {
		return nil, err
	}

	// 2. Cek apakah sudah absen
	existing, _ := u.attendanceRepo.GetByWorkerAndDate(ctx, input.WorkerID, cleanDate)
	if existing != nil {
		return nil, domain.NewError(domain.ErrConflict, "Absensi pekerja untuk tanggal ini sudah dicatat sebelumnya")
	}

	var manualIdx *float64
	if input.WorkDurationIndex > 0 {
		manualIdx = &input.WorkDurationIndex
	}
	durationIndex := calculateDurationIndex(input.Status, manualIdx)
	totalAmount := durationIndex * worker.DailyRate

	att := &domain.Attendance{
		WorkerID:          input.WorkerID,
		AttendanceDate:    cleanDate,
		Status:            input.Status,
		WorkDurationIndex: durationIndex,
		DailyRate:         worker.DailyRate,
		TotalAmount:       totalAmount,
		Notes:             input.Notes,
		CreatedByID:       input.CreatedByID,
	}

	if err := u.attendanceRepo.Create(ctx, att); err != nil {
		return nil, err
	}

	return att, nil
}

// RECORD BATCH ATTENDANCE (Absensi Masal Harian Seluruh Karyawan)
func (u *attendanceUsecase) RecordBatchAttendance(c context.Context, input domain.BatchAttendanceInput) ([]domain.Attendance, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.AttendanceDate.IsZero() {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tanggal absensi tidak valid")
	}
	if len(input.Items) == 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Daftar absensi pekerja tidak boleh kosong")
	}

	// Format tanggal murni tanpa offset jam (YYYY-MM-DD)
	cleanDate := time.Date(
		input.AttendanceDate.Year(),
		input.AttendanceDate.Month(),
		input.AttendanceDate.Day(),
		0, 0, 0, 0, time.UTC,
	)

	var attendancesToCreate []domain.Attendance
	processedWorkers := make(map[string]bool) // 👈 Map untuk cegah worker_id ganda di payload yang sama

	for _, item := range input.Items {
		if item.WorkerID == "" {
			continue
		}

		// Jika worker_id terduplikasi dalam 1 request batch, skip
		if processedWorkers[item.WorkerID] {
			continue
		}

		// Check ke database apakah pekerja ini sudah absen di tanggal tersebut
		existing, _ := u.attendanceRepo.GetByWorkerAndDate(ctx, item.WorkerID, cleanDate)
		if existing != nil {
			continue
		}

		worker, err := u.workerRepo.GetByID(ctx, item.WorkerID)
		if err != nil {
			continue
		}

		durationIndex := calculateDurationIndex(item.Status, item.WorkDurationIndex)
		totalAmount := durationIndex * worker.DailyRate

		attendancesToCreate = append(attendancesToCreate, domain.Attendance{
			WorkerID:          item.WorkerID,
			AttendanceDate:    cleanDate,
			Status:            item.Status,
			WorkDurationIndex: durationIndex,
			DailyRate:         worker.DailyRate,
			TotalAmount:       totalAmount,
			Notes:             item.Notes,
			CreatedByID:       input.CreatedByID,
		})

		processedWorkers[item.WorkerID] = true
	}

	if len(attendancesToCreate) == 0 {
		return nil, domain.NewError(domain.ErrConflict, "Seluruh pekerja yang dipilih sudah mencatat absensi pada tanggal ini")
	}

	if err := u.attendanceRepo.CreateBatch(ctx, attendancesToCreate); err != nil {
		return nil, err
	}

	return attendancesToCreate, nil
}

func (u *attendanceUsecase) GetAttendance(c context.Context, id string) (*domain.Attendance, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID Absensi tidak valid")
	}

	return u.attendanceRepo.GetByID(ctx, id)
}

func (u *attendanceUsecase) ListAttendances(c context.Context, query domain.PaginationQuery, filter domain.AttendanceFilter) ([]domain.Attendance, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	attendances, total, err := u.attendanceRepo.Fetch(ctx, filter, query.Limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       query.Limit,
		TotalItems:  total,
		TotalPages:  int(math.Ceil(float64(total) / float64(query.Limit))),
	}

	return attendances, meta, nil
}

func (u *attendanceUsecase) UpdateAttendance(c context.Context, id string, input domain.AttendanceUpdateInput) (*domain.Attendance, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID Absensi tidak valid")
	}

	existing, err := u.attendanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Lock jika sudah masuk ke Payroll yang Lunas
	if existing.PayrollID != nil && *existing.PayrollID != "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Absensi ini sudah masuk ke rekap penggajian dan tidak dapat diubah")
	}

	if input.Status != "" {
		existing.Status = input.Status
	}

	existing.WorkDurationIndex = calculateDurationIndex(existing.Status, input.WorkDurationIndex)
	existing.TotalAmount = existing.WorkDurationIndex * existing.DailyRate
	existing.Notes = input.Notes
	existing.UpdatedAt = time.Now()

	if err := u.attendanceRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (u *attendanceUsecase) DeleteAttendance(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return domain.NewError(domain.ErrBadParamInput, "ID Absensi tidak valid")
	}

	existing, err := u.attendanceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.PayrollID != nil && *existing.PayrollID != "" {
		return domain.NewError(domain.ErrBadParamInput, "Absensi ini sudah masuk ke rekap penggajian dan tidak dapat dihapus")
	}

	return u.attendanceRepo.Delete(ctx, id)
}
