package common

import "math"

func GetTrustQuota() int {
	return int(10 * QuotaPerUnit)
}

// QuotaUpperLimit 返回 limitUnits 单位（美元）对应的额度上限。
// 乘积先按 float64 计算并钳制到当前平台 int 最大值：直接写
// int(limitUnits * QuotaPerUnit) 在 32 位平台（int 为 32 位）会溢出为
// 负数，使"超过上限"校验恒成立、合法额度被全量拒绝。
func QuotaUpperLimit(limitUnits float64) int {
	limit := limitUnits * QuotaPerUnit
	// float64(math.MaxInt) 在 64 位平台舍入为 2^63（已超出 MaxInt），
	// 用其下最近可表示值作钳制边界，避免 [2^63-1024, 2^63) 区间漏检。
	if limit >= math.Nextafter(float64(math.MaxInt), 0) {
		return math.MaxInt
	}
	return int(limit)
}
