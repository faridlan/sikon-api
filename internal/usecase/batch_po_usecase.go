package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type batchPoUsecase struct {
	batchPoRepo    domain.BatchPORepository
	contextTimeout time.Duration
}

func NewBatchPOUsecase(bpr domain.BatchPORepository, timeout time.Duration) domain.BatchPOUsecase {
	return &batchPoUsecase{
		batchPoRepo:    bpr,
		contextTimeout: timeout,
	}
}

func (u *batchPoUsecase) CreateBatchPO(c context.Context, input domain.BatchPOCreateInput) (*domain.BatchPO, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.OpenDate.IsZero() || input.CloseDate.IsZero() {
		return nil, domain.NewError(domain.ErrBadParamInput, "open_date dan close_date wajib diisi")
	}
	if input.OpenDate.After(input.CloseDate) {
		return nil, domain.NewError(domain.ErrBadParamInput, "open_date tidak boleh setelah close_date")
	}
	// Aturan bisnis: close_date = tanggal mulai kerja. Tidak dipaksa sama persis (biar ada
	// ruang buat kasus khusus), tapi kalau close_date > start_date, kemungkinan besar salah input.
	if input.CloseDate.After(input.StartDate) {
		return nil, domain.NewError(domain.ErrBadParamInput, "close_date seharusnya tidak melewati start_date (tanggal mulai kerja)")
	}

	batchPO := &domain.BatchPO{
		Name:        input.Name,
		TargetMonth: input.TargetMonth,
		TargetYear:  input.TargetYear,
		OpenDate:    input.OpenDate,
		CloseDate:   input.CloseDate,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
		Quota:       input.Quota,
		Status:      domain.BatchPOStatusDraft, // Otomatis berstatus draft saat pertama kali dibuat
	}

	if err := u.batchPoRepo.Create(ctx, batchPO); err != nil {
		return nil, err
	}

	return batchPO, nil
}

func (u *batchPoUsecase) GetBatchPO(c context.Context, id string) (*domain.BatchPO, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	batchPO, err := u.batchPoRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Batch PO tidak ditemukan")
		}
		return nil, err
	}
	return batchPO, nil
}

func (u *batchPoUsecase) ListBatchPOs(c context.Context, query domain.PaginationQuery) ([]domain.BatchPO, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	batchPOs, totalItems, err := u.batchPoRepo.Fetch(ctx, limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	return batchPOs, meta, nil
}

func (u *batchPoUsecase) ListActiveBatchPOs(c context.Context) ([]domain.BatchPO, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.batchPoRepo.FetchActive(ctx)
}

// GetSuggestedOpenDate mengembalikan close_date dari PO terakhir yang ada, untuk
// di-prefill FE sebagai open_date PO baru (sesuai aturan: buka PO baru = tutup PO sebelumnya).
// Mengembalikan nil kalau belum ada PO sama sekali (Admin isi manual untuk PO pertama).
func (u *batchPoUsecase) GetSuggestedOpenDate(c context.Context) (*time.Time, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	latest, err := u.batchPoRepo.FetchLatest(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil // Belum ada PO sama sekali, bukan error
		}
		return nil, err
	}

	return &latest.CloseDate, nil
}

func (u *batchPoUsecase) UpdateBatchPO(c context.Context, id string, input domain.BatchPOUpdateInput) (*domain.BatchPO, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	batchPO, err := u.batchPoRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Batch PO tidak ditemukan")
		}
		return nil, err
	}

	if input.Name != "" {
		batchPO.Name = input.Name
	}
	if input.TargetMonth != nil {
		batchPO.TargetMonth = *input.TargetMonth
	}
	if input.TargetYear != nil {
		batchPO.TargetYear = *input.TargetYear
	}
	if input.OpenDate != nil {
		batchPO.OpenDate = *input.OpenDate
	}
	if input.CloseDate != nil {
		batchPO.CloseDate = *input.CloseDate
	}
	if input.StartDate != nil {
		batchPO.StartDate = *input.StartDate
	}
	if input.EndDate != nil {
		batchPO.EndDate = *input.EndDate
	}
	if input.Quota != nil {
		batchPO.Quota = *input.Quota
	}

	if batchPO.OpenDate.After(batchPO.CloseDate) {
		return nil, domain.NewError(domain.ErrBadParamInput, "open_date tidak boleh setelah close_date")
	}

	if err := u.batchPoRepo.Update(ctx, batchPO); err != nil {
		return nil, err
	}
	return batchPO, nil
}

func (u *batchPoUsecase) UpdateStatus(c context.Context, id string, status domain.BatchPOStatus) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if !status.IsValid() {
		return domain.NewError(domain.ErrBadParamInput, "Status PO tidak valid")
	}

	batchPO, err := u.batchPoRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Batch PO tidak ditemukan")
		}
		return err
	}

	batchPO.Status = status
	return u.batchPoRepo.Update(ctx, batchPO)
}

func (u *batchPoUsecase) DeleteBatchPO(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	_, err := u.batchPoRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Batch PO tidak ditemukan")
		}
		return err
	}

	return u.batchPoRepo.Delete(ctx, id)
}
