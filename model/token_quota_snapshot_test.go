package model

import "testing"

func TestTokenQuotaSnapshotPermanent(t *testing.T) {
	token := &Token{
		QuotaType:   1,
		RemainQuota: 700,
		UsedQuota:   300,
	}

	snapshot := token.getQuotaSnapshot(0)
	if snapshot.QuotaType != 1 {
		t.Fatalf("expected quota type 1, got %d", snapshot.QuotaType)
	}
	if snapshot.TotalGranted != 1000 {
		t.Fatalf("expected total granted 1000, got %d", snapshot.TotalGranted)
	}
	if snapshot.TotalUsed != 300 {
		t.Fatalf("expected total used 300, got %d", snapshot.TotalUsed)
	}
	if snapshot.TotalAvailable != 700 {
		t.Fatalf("expected total available 700, got %d", snapshot.TotalAvailable)
	}
}

func TestTokenQuotaSnapshotLegacyPermanent(t *testing.T) {
	token := &Token{
		QuotaType:      0,
		UnlimitedQuota: false,
		RemainQuota:    800,
		UsedQuota:      200,
	}

	snapshot := token.getQuotaSnapshot(0)
	if snapshot.QuotaType != 1 {
		t.Fatalf("expected legacy quota type to map to 1, got %d", snapshot.QuotaType)
	}
	if snapshot.TotalGranted != 1000 {
		t.Fatalf("expected total granted 1000, got %d", snapshot.TotalGranted)
	}
	if snapshot.TotalUsed != 200 {
		t.Fatalf("expected total used 200, got %d", snapshot.TotalUsed)
	}
	if snapshot.TotalAvailable != 800 {
		t.Fatalf("expected total available 800, got %d", snapshot.TotalAvailable)
	}
}

func TestTokenQuotaSnapshotWindowResetsForFeedback(t *testing.T) {
	token := &Token{
		QuotaType:       2,
		WindowHours:     12,
		WindowQuota:     1000,
		WindowStartHour: 0,
		WindowStartTime: 1,
		WindowUsedQuota: 900,
	}

	snapshot := token.getQuotaSnapshot(13 * 3600)
	if snapshot.QuotaType != 2 {
		t.Fatalf("expected quota type 2, got %d", snapshot.QuotaType)
	}
	if snapshot.TotalGranted != 1000 {
		t.Fatalf("expected total granted 1000, got %d", snapshot.TotalGranted)
	}
	if snapshot.TotalUsed != 0 {
		t.Fatalf("expected total used 0 after reset, got %d", snapshot.TotalUsed)
	}
	if snapshot.TotalAvailable != 1000 {
		t.Fatalf("expected total available 1000 after reset, got %d", snapshot.TotalAvailable)
	}
}

func TestTokenQuotaSnapshotWindowCycleUsesTighterLimit(t *testing.T) {
	token := &Token{
		QuotaType:       3,
		WindowHours:     12,
		WindowQuota:     100,
		WindowStartHour: 0,
		WindowStartTime: 13 * 24 * 3600,
		WindowUsedQuota: 40,
		CycleDays:       7,
		CycleQuota:      500,
		CycleStartTime:  8 * 24 * 3600,
		CycleUsedQuota:  100,
	}

	snapshot := token.getQuotaSnapshot(13*24*3600 + 1)
	if snapshot.QuotaType != 3 {
		t.Fatalf("expected quota type 3, got %d", snapshot.QuotaType)
	}
	if snapshot.TotalGranted != 100 {
		t.Fatalf("expected total granted 100 from window quota, got %d", snapshot.TotalGranted)
	}
	if snapshot.TotalUsed != 40 {
		t.Fatalf("expected total used 40 from window quota, got %d", snapshot.TotalUsed)
	}
	if snapshot.TotalAvailable != 60 {
		t.Fatalf("expected total available 60 from window quota, got %d", snapshot.TotalAvailable)
	}
}

func TestTokenQuotaSnapshotWindowCycleFallsBackToCycleLimit(t *testing.T) {
	token := &Token{
		QuotaType:       3,
		WindowHours:     12,
		WindowQuota:     300,
		WindowStartHour: 0,
		WindowStartTime: 13 * 24 * 3600,
		WindowUsedQuota: 40,
		CycleDays:       7,
		CycleQuota:      100,
		CycleStartTime:  8 * 24 * 3600,
		CycleUsedQuota:  70,
	}

	snapshot := token.getQuotaSnapshot(13*24*3600 + 1)
	if snapshot.TotalGranted != 100 {
		t.Fatalf("expected total granted 100 from cycle quota, got %d", snapshot.TotalGranted)
	}
	if snapshot.TotalUsed != 70 {
		t.Fatalf("expected total used 70 from cycle quota, got %d", snapshot.TotalUsed)
	}
	if snapshot.TotalAvailable != 30 {
		t.Fatalf("expected total available 30 from cycle quota, got %d", snapshot.TotalAvailable)
	}
}

func TestTokenQuotaSnapshotClampsNegativeUsedQuota(t *testing.T) {
	token := &Token{
		QuotaType:       3,
		WindowHours:     12,
		WindowQuota:     100,
		WindowStartHour: 0,
		WindowStartTime: 13 * 24 * 3600,
		WindowUsedQuota: -20,
		CycleDays:       7,
		CycleQuota:      300,
		CycleStartTime:  8 * 24 * 3600,
		CycleUsedQuota:  -50,
	}

	snapshot := token.getQuotaSnapshot(13*24*3600 + 1)
	if snapshot.TotalUsed != 0 {
		t.Fatalf("expected total used 0 after clamping negative counters, got %d", snapshot.TotalUsed)
	}
	if snapshot.TotalAvailable != 100 {
		t.Fatalf("expected total available 100 after clamping negative counters, got %d", snapshot.TotalAvailable)
	}
}
