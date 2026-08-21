package common

import (
	"math"
	"strconv"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDecimalToIntTruncatesFractionLikeIntPart(t *testing.T) {
	got, err := DecimalToInt(decimal.NewFromFloat(10.9))
	if err != nil {
		t.Fatalf("DecimalToInt(10.9) error = %v", err)
	}
	if got != 10 {
		t.Fatalf("DecimalToInt(10.9) = %d, want 10 (keep IntPart truncation semantics)", got)
	}
}

// TestDecimalToIntInt32PlatformBoundary 复现审查场景：$8599 × QuotaPerUnit(500000)
// = 4,299,500,000。64 位构建可正常表示；32 位构建直接 int(IntPart()) 会回绕成
// 4,532,704（静默少入账），DecimalToInt 必须显式报错。
func TestDecimalToIntInt32PlatformBoundary(t *testing.T) {
	d := decimal.NewFromInt(8599).Mul(decimal.NewFromFloat(500000))
	got, err := DecimalToInt(d)
	if strconv.IntSize == 32 {
		if err == nil {
			t.Fatalf("DecimalToInt(%s) = %d on 32-bit platform, want error (was silently wrapped to 4532704 by int(IntPart()))", d.String(), got)
		}
		return
	}
	if err != nil {
		t.Fatalf("DecimalToInt(%s) error = %v on 64-bit platform", d.String(), err)
	}
	// 用 int64 比较，避免 4299500000 字面量在 32 位构建下默认类型溢出无法编译。
	if int64(got) != int64(4299500000) {
		t.Fatalf("DecimalToInt(%s) = %d, want 4299500000", d.String(), got)
	}
}

// TestDecimalToIntRejectsInt64Overflow 验证超过 int64 的乘积（IntPart 自身已回绕，
// 实测 4×5e18 的 IntPart()=1,553,255,926,290,448,384）在两种字宽下都被拒绝。
func TestDecimalToIntRejectsInt64Overflow(t *testing.T) {
	for _, s := range []string{"10000000000000000000", "-10000000000000000000"} {
		d, parseErr := decimal.NewFromString(s)
		if parseErr != nil {
			t.Fatalf("decimal.NewFromString(%s) error = %v", s, parseErr)
		}
		if got, err := DecimalToInt(d); err == nil {
			t.Fatalf("DecimalToInt(%s) = %d, want error on %d-bit platform", s, got, strconv.IntSize)
		}
	}
}

// TestDecimalToIntBoundaryInclusive 验证恰好等于 math.MaxInt 的值放行（比较先于取整，
// 不会因截断误杀边界值）。
func TestDecimalToIntBoundaryInclusive(t *testing.T) {
	d := decimal.NewFromInt(math.MaxInt)
	got, err := DecimalToInt(d)
	if err != nil {
		t.Fatalf("DecimalToInt(maxInt) error = %v", err)
	}
	if got != math.MaxInt {
		t.Fatalf("DecimalToInt(maxInt) = %d, want %d", got, math.MaxInt)
	}
}
