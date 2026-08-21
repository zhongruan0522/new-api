package common

import (
	"math"
	"testing"
)

func TestQuotaUpperLimitPositiveOnAllPlatforms(t *testing.T) {
	// 回归：int(limitUnits * QuotaPerUnit) 在 32 位平台溢出为负，
	// 使 token/user 的额度上限校验恒拒绝合法值。
	got := QuotaUpperLimit(1000000000)
	if got <= 0 {
		t.Fatalf("QuotaUpperLimit = %d, want > 0", got)
	}
	if got > math.MaxInt {
		t.Fatalf("QuotaUpperLimit = %d, want <= MaxInt(%d)", got, math.MaxInt)
	}
}

func TestQuotaUpperLimitClampsToMaxInt(t *testing.T) {
	old := QuotaPerUnit
	defer func() { QuotaPerUnit = old }()

	QuotaPerUnit = 1e18
	if got := QuotaUpperLimit(1e9); got != math.MaxInt {
		t.Fatalf("QuotaUpperLimit = %d, want MaxInt(%d)", got, math.MaxInt)
	}

	// 恰好落在 float64(MaxInt) 舍入边界上的值也必须钳制，
	// 不得进入越界 int() 转换（64 位平台窗口为 [2^63-1024, 2^63)）。
	boundary := math.Nextafter(float64(math.MaxInt), 0)
	QuotaPerUnit = boundary / 2
	if got := QuotaUpperLimit(2); got != math.MaxInt {
		t.Fatalf("QuotaUpperLimit = %d, want MaxInt(%d)", got, math.MaxInt)
	}
}
