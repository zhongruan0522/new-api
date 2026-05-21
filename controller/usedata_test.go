package controller

import "testing"

func TestIsUserQuotaRangeTooLongAllowsOneMonthWindow(t *testing.T) {
	start := int64(1_700_000_000)
	end := start + maxUserQuotaRangeSeconds

	if isUserQuotaRangeTooLong(start, end) {
		t.Fatalf("expected %d-second range to be allowed", maxUserQuotaRangeSeconds)
	}
}

func TestIsUserQuotaRangeTooLongRejectsOverOneMonthWindow(t *testing.T) {
	start := int64(1_700_000_000)
	end := start + maxUserQuotaRangeSeconds + 1

	if !isUserQuotaRangeTooLong(start, end) {
		t.Fatalf("expected range longer than %d seconds to be rejected", maxUserQuotaRangeSeconds)
	}
}

func TestIsUserQuotaRangeTooLongIgnoresInvalidRange(t *testing.T) {
	if isUserQuotaRangeTooLong(0, 0) {
		t.Fatal("expected empty range to be ignored by span validation")
	}
	if isUserQuotaRangeTooLong(200, 100) {
		t.Fatal("expected reversed range to be ignored by span validation")
	}
}
