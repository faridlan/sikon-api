package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type workLogUsecase struct {
	workLogRepo    domain.WorkLogRepository
	workerRepo     domain.WorkerRepository
	orderRepo      domain.OrderRepository
	batchPORepo    domain.BatchPORepository
	contextTimeout time.Duration
}

func NewWorkLogUsecase(
	repo domain.WorkLogRepository,
	workerRepo domain.WorkerRepository,
	orderRepo domain.OrderRepository,
	batchPORepo domain.BatchPORepository,
	timeout time.Duration,
) domain.WorkLogUsecase {
	return &workLogUsecase{
		workLogRepo:    repo,
		workerRepo:     workerRepo,
		orderRepo:      orderRepo,
		batchPORepo:    batchPORepo,
		contextTimeout: timeout,
	}
}

func (u *workLogUsecase) CreateWorkLog(c context.Context, input domain.WorkLogCreateInput) (*domain.WorkLog, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.WorkerID == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Worker ID harus diisi")
	}
	if input.Qty <= 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Qty harus lebih besar dari 0")
	}
	if input.RatePerQty <= 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tarif per pcs harus lebih besar dari 0")
	}
	if input.WorkDate.IsZero() {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tanggal kerja tidak valid")
	}

	// Pastikan worker ada
	_, err := u.workerRepo.GetByID(ctx, input.WorkerID)
	if err != nil {
		return nil, err
	}

	// 🚨 GEMBOK VALIDASI QTY PER ORDER KONSUMEN
	if input.OrderID != nil && *input.OrderID != "" {
		order, err := u.orderRepo.GetByID(ctx, *input.OrderID)
		if err != nil {
			return nil, err
		}

		existingQty, err := u.workLogRepo.GetTotalQtyByOrderAndJobType(ctx, *input.OrderID, input.JobType, "")
		if err != nil {
			return nil, err
		}

		totalProposedQty := existingQty + input.Qty
		if totalProposedQty > order.TotalQty {
			sisaKuota := order.TotalQty - existingQty
			return nil, domain.NewError(domain.ErrConflict, fmt.Sprintf("Gagal catat borongan %s: Total Qty (%d pcs) melebihi Qty Pesanan Konsumen (%d pcs). Sisa kuota: %d pcs.", input.JobType, totalProposedQty, order.TotalQty, sisaKuota))
		}
	}

	totalAmount := float64(input.Qty) * input.RatePerQty

	log := &domain.WorkLog{
		WorkerID:    input.WorkerID,
		BatchPoID:   input.BatchPoID,
		OrderID:     input.OrderID,
		JobType:     input.JobType,
		Qty:         input.Qty,
		RatePerQty:  input.RatePerQty,
		TotalAmount: totalAmount,
		WorkDate:    input.WorkDate,
		Notes:       input.Notes,
		CreatedByID: input.CreatedByID,
	}

	if err := u.workLogRepo.Create(ctx, log); err != nil {
		return nil, err
	}

	return log, nil
}

// 🚨 FITUR BARU: AUTO DISTRIBUTE WORKLOAD PER BATCH PO
func (u *workLogUsecase) DistributeWorkLoad(c context.Context, input domain.DistributeWorkLoadInput) ([]domain.WorkLog, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.BatchPOID == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Batch PO ID harus diisi")
	}
	if len(input.WorkerIDs) == 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Pilih minimal 1 pekerja untuk pembagian tugas")
	}
	if input.RatePerQty <= 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tarif per pcs harus lebih besar dari 0")
	}
	if input.WorkDate.IsZero() {
		input.WorkDate = time.Now()
	}

	// 1. Ambil seluruh order aktif yang terikat dengan Batch PO ini
	orders, err := u.orderRepo.GetByBatchPOID(ctx, input.BatchPOID)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, domain.NewError(domain.ErrNotFound, "Tidak ada order/konsumen terdaftar pada Batch PO ini")
	}

	// 2. Siapkan antrean pengerjaan per order berdasarkan sisa Qty yang belum dialokasikan
	type orderTask struct {
		OrderID     string
		OrderNumber string
		Remaining   int
	}

	var tasks []orderTask
	var totalBatchRemaining int

	for _, ord := range orders {
		allocatedQty, err := u.workLogRepo.GetTotalQtyByOrderAndJobType(ctx, ord.ID, input.JobType, "")
		if err != nil {
			return nil, err
		}

		remaining := ord.TotalQty - allocatedQty
		if remaining > 0 {
			tasks = append(tasks, orderTask{
				OrderID:     ord.ID,
				OrderNumber: ord.OrderNumber,
				Remaining:   remaining,
			})
			totalBatchRemaining += remaining
		}
	}

	if totalBatchRemaining <= 0 {
		return nil, domain.NewError(domain.ErrConflict, fmt.Sprintf("Seluruh pesanan pada Batch PO ini sudah dialokasikan sepenuhnya untuk jenis pekerjaan '%s'", input.JobType))
	}

	// 3. Algoritma Pembagian Rata ke Pekerja
	numWorkers := len(input.WorkerIDs)
	baseQtyPerWorker := totalBatchRemaining / numWorkers
	extraQty := totalBatchRemaining % numWorkers

	// Hitung kuota target yang akan didapatkan masing-masing pekerja
	workerTargets := make(map[string]int)
	for i, wID := range input.WorkerIDs {
		target := baseQtyPerWorker
		if i < extraQty {
			target++ // Sisa pembagian didistribusikan ke pekerja awal
		}
		workerTargets[wID] = target
	}

	// 4. Distribusikan task ke pekerja secara terpresisi per order konsumen
	var logsToCreate []domain.WorkLog
	workerIdx := 0

	for _, task := range tasks {
		taskRemaining := task.Remaining

		for taskRemaining > 0 && workerIdx < numWorkers {
			currentWorkerID := input.WorkerIDs[workerIdx]
			workerRemainingCapacity := workerTargets[currentWorkerID]

			if workerRemainingCapacity == 0 {
				workerIdx++
				continue
			}

			// Tentukan Qty borongan untuk pekerja ini di order ini
			assignQty := taskRemaining
			if assignQty > workerRemainingCapacity {
				assignQty = workerRemainingCapacity
			}

			batchPoID := input.BatchPOID
			orderID := task.OrderID
			totalAmount := float64(assignQty) * input.RatePerQty

			logsToCreate = append(logsToCreate, domain.WorkLog{
				WorkerID:    currentWorkerID,
				BatchPoID:   &batchPoID,
				OrderID:     &orderID,
				JobType:     input.JobType,
				Qty:         assignQty,
				RatePerQty:  input.RatePerQty,
				TotalAmount: totalAmount,
				WorkDate:    input.WorkDate,
				Notes:       fmt.Sprintf("%s (Auto Distribute %s)", input.Notes, task.OrderNumber),
				CreatedByID: input.CreatedByID,
			})

			taskRemaining -= assignQty
			workerTargets[currentWorkerID] -= assignQty

			if workerTargets[currentWorkerID] == 0 {
				workerIdx++
			}
		}
	}

	// 5. Simpan seluruh WorkLog secara batch
	if len(logsToCreate) > 0 {
		if err := u.workLogRepo.CreateBatch(ctx, logsToCreate); err != nil {
			return nil, err
		}
	}

	return logsToCreate, nil
}

func (u *workLogUsecase) GetWorkLog(c context.Context, id string) (*domain.WorkLog, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID WorkLog tidak valid")
	}

	return u.workLogRepo.GetByID(ctx, id)
}

func (u *workLogUsecase) ListWorkLogs(c context.Context, query domain.PaginationQuery, filter domain.WorkLogFilter) ([]domain.WorkLog, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	logs, total, err := u.workLogRepo.Fetch(ctx, filter, query.Limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       query.Limit,
		TotalItems:  total,
		TotalPages:  int(math.Ceil(float64(total) / float64(query.Limit))),
	}

	return logs, meta, nil
}

func (u *workLogUsecase) UpdateWorkLog(c context.Context, id string, input domain.WorkLogUpdateInput) (*domain.WorkLog, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID WorkLog tidak valid")
	}

	existing, err := u.workLogRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.PayrollID != nil && *existing.PayrollID != "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Catatan borongan ini sudah masuk ke rekap penggajian dan tidak dapat diubah")
	}

	targetOrderID := existing.OrderID
	if input.OrderID != nil {
		targetOrderID = input.OrderID
	}

	targetJobType := existing.JobType
	if input.JobType != "" {
		targetJobType = input.JobType
	}

	targetQty := existing.Qty
	if input.Qty > 0 {
		targetQty = input.Qty
	}

	// 🚨 VALIDASI GEMBOK QTY OVERLOAD SAAT UPDATE
	if targetOrderID != nil && *targetOrderID != "" {
		order, err := u.orderRepo.GetByID(ctx, *targetOrderID)
		if err != nil {
			return nil, err
		}

		existingTotalQty, err := u.workLogRepo.GetTotalQtyByOrderAndJobType(ctx, *targetOrderID, targetJobType, id)
		if err != nil {
			return nil, err
		}

		if (existingTotalQty + targetQty) > order.TotalQty {
			sisa := order.TotalQty - existingTotalQty
			return nil, domain.NewError(domain.ErrConflict, fmt.Sprintf("Gagal update borongan: Total Qty (%d pcs) melebihi Qty Pesanan Konsumen (%d pcs). Sisa kuota: %d pcs.", existingTotalQty+targetQty, order.TotalQty, sisa))
		}
	}

	if input.WorkerID != "" {
		existing.WorkerID = input.WorkerID
	}
	if input.BatchPoID != nil {
		existing.BatchPoID = input.BatchPoID
	}
	if input.OrderID != nil {
		existing.OrderID = input.OrderID
	}
	if input.JobType != "" {
		existing.JobType = input.JobType
	}
	if input.Qty > 0 {
		existing.Qty = input.Qty
	}
	if input.RatePerQty > 0 {
		existing.RatePerQty = input.RatePerQty
	}
	if !input.WorkDate.IsZero() {
		existing.WorkDate = input.WorkDate
	}
	existing.Notes = input.Notes

	existing.TotalAmount = float64(existing.Qty) * existing.RatePerQty
	existing.UpdatedAt = time.Now()

	if err := u.workLogRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (u *workLogUsecase) DeleteWorkLog(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return domain.NewError(domain.ErrBadParamInput, "ID WorkLog tidak valid")
	}

	existing, err := u.workLogRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.PayrollID != nil && *existing.PayrollID != "" {
		return domain.NewError(domain.ErrBadParamInput, "Catatan borongan ini sudah masuk ke rekap penggajian dan tidak dapat dihapus")
	}

	return u.workLogRepo.Delete(ctx, id)
}
