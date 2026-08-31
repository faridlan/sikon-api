package postgres

import (
	"context"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type attendanceRepository struct {
	db *gorm.DB
}

func NewAttendanceRepository(db *gorm.DB) domain.AttendanceRepository {
	return &attendanceRepository{db: db}
}

func (r *attendanceRepository) Create(ctx context.Context, att *domain.Attendance) error {
	model := FromAttendanceDomain(att)

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "worker_id"},
				{Name: "attendance_date"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"status",
				"work_duration_index",
				"daily_rate",
				"total_amount",
				"notes",
				"created_by",
				"updated_at",
			}),
		}).
		Create(model).Error

	if err != nil {
		return TranslateError(err)
	}

	att.ID = model.ID
	att.CreatedAt = model.CreatedAt
	att.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *attendanceRepository) CreateBatch(ctx context.Context, attendances []domain.Attendance) error {
	if len(attendances) == 0 {
		return nil
	}

	var models []AttendanceModel
	for _, a := range attendances {
		models = append(models, *FromAttendanceDomain(&a))
	}

	// 🚨 KUNCI PERBAIKAN: Gunakan ON CONFLICT DO UPDATE
	// Jika worker_id + attendance_date sudah ada di DB, timpa status & nilainya
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "worker_id"},
				{Name: "attendance_date"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"status",
				"work_duration_index",
				"daily_rate",
				"total_amount",
				"notes",
				"created_by",
				"updated_at",
			}),
		}).
		Create(&models).Error

	if err != nil {
		return TranslateError(err)
	}

	return nil
}

func (r *attendanceRepository) GetByID(ctx context.Context, id string) (*domain.Attendance, error) {
	var model AttendanceModel
	err := r.db.WithContext(ctx).
		Preload("Worker").
		Preload("Creator").
		Where("id = ?", id).
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *attendanceRepository) GetByWorkerAndDate(ctx context.Context, workerID string, date time.Time) (*domain.Attendance, error) {
	var model AttendanceModel

	// Gunakan DATE(attendance_date) untuk mencocokkan tanggal murni YYYY-MM-DD
	formattedDate := date.Format("2006-01-02")
	err := r.db.WithContext(ctx).
		Where("worker_id = ? AND DATE(attendance_date) = ?", workerID, formattedDate).
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *attendanceRepository) Fetch(ctx context.Context, filter domain.AttendanceFilter, limit, offset int) ([]domain.Attendance, int64, error) {
	var models []AttendanceModel
	var total int64

	query := r.db.WithContext(ctx).Model(&AttendanceModel{})

	if filter.WorkerID != nil && *filter.WorkerID != "" {
		query = query.Where("worker_id = ?", *filter.WorkerID)
	}

	if filter.PayrollID != nil && *filter.PayrollID != "" {
		query = query.Where("payroll_id = ?", *filter.PayrollID)
	}

	if filter.IsUnpaid {
		query = query.Where("payroll_id IS NULL")
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.StartDate != nil && filter.EndDate != nil {
		query = query.Where("attendance_date >= ? AND attendance_date <= ?", filter.StartDate, filter.EndDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	err := query.
		Preload("Worker").
		Preload("Creator").
		Order("attendance_date DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error

	if err != nil {
		return nil, 0, TranslateError(err)
	}

	var results []domain.Attendance
	for _, m := range models {
		results = append(results, *m.ToDomain())
	}

	return results, total, nil
}

func (r *attendanceRepository) Update(ctx context.Context, att *domain.Attendance) error {
	model := FromAttendanceDomain(att)

	result := r.db.WithContext(ctx).
		Model(&AttendanceModel{}).
		Where("id = ?", att.ID).
		Updates(map[string]interface{}{
			"status":              model.Status,
			"work_duration_index": model.WorkDurationIndex,
			"daily_rate":          model.DailyRate,
			"total_amount":        model.TotalAmount,
			"notes":               model.Notes,
			"updated_at":          model.UpdatedAt,
		})

	if result.Error != nil {
		return TranslateError(result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *attendanceRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&AttendanceModel{})
	if result.Error != nil {
		return TranslateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *attendanceRepository) GetTotalAmountByWorkerAndPeriod(ctx context.Context, workerID string, startDate, endDate time.Time) (float64, int, error) {
	type result struct {
		TotalAmount float64
		TotalDays   int
	}

	var res result
	err := r.db.WithContext(ctx).Model(&AttendanceModel{}).
		Where("worker_id = ? AND attendance_date >= ? AND attendance_date <= ? AND deleted_at IS NULL", workerID, startDate, endDate).
		Select("COALESCE(SUM(total_amount), 0) as total_amount, COUNT(id) as total_days").
		Scan(&res).Error

	if err != nil {
		return 0, 0, TranslateError(err)
	}

	return res.TotalAmount, res.TotalDays, nil
}
