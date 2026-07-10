package postgres

import "testing"

func TestCalculateRemainingQuota(t *testing.T) {
	tests := []struct {
		name    string
		quota   int
		usedQty int64
		want    int64
	}{
		{name: "positive remaining", quota: 100, usedQty: 30, want: 70},
		{name: "exhausted quota", quota: 10, usedQty: 12, want: 0},
		{name: "unlimited quota", quota: 0, usedQty: 5, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateRemainingQuota(tt.quota, tt.usedQty)
			if got != tt.want {
				t.Fatalf("calculateRemainingQuota(%d, %d) = %d, want %d", tt.quota, tt.usedQty, got, tt.want)
			}
		})
	}
}
