package dto

import (
	"testing"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestToOrderResponse_IncludesBatchPO(t *testing.T) {
	batchPO := &domain.BatchPO{
		ID:     "batch-po-123",
		Name:   "PO Januari",
		Status: domain.BatchPOStatusActive,
	}

	order := &domain.Order{
		ID:        "order-123",
		BatchPoID: batchPO.ID,
		BatchPO:   batchPO,
	}

	resp := ToOrderResponse(order)

	assert.NotNil(t, resp.BatchPO)
	assert.Equal(t, batchPO.ID, resp.BatchPO.ID)
	assert.Equal(t, batchPO.Name, resp.BatchPO.Name)
	assert.Equal(t, string(batchPO.Status), resp.BatchPO.Status)
}
