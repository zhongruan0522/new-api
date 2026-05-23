package model

import "testing"

func TestUserSubscriptionSnapshotDoesNotDoubleCountWindowUsage(t *testing.T) {
	sub := &UserSubscription{
		StartsAt:        0,
		NextResetAt:     3600,
		TotalQuota:      500,
		ResetQuota:      500,
		UsedTotalQuota:  200,
		WindowUsedQuota: 200,
	}

	snapshot := sub.Snapshot(100)
	if snapshot.TotalRemaining != 300 {
		t.Fatalf("total remaining = %d, want 300", snapshot.TotalRemaining)
	}
	if snapshot.EffectiveResetCap != 500 {
		t.Fatalf("effective reset cap = %d, want 500", snapshot.EffectiveResetCap)
	}
	if snapshot.WindowRemaining != 300 {
		t.Fatalf("window remaining = %d, want 300", snapshot.WindowRemaining)
	}
}

func TestUserSubscriptionSnapshotCapsWindowByTotalRemaining(t *testing.T) {
	sub := &UserSubscription{
		StartsAt:        0,
		NextResetAt:     3600,
		TotalQuota:      1000,
		ResetQuota:      500,
		UsedTotalQuota:  900,
		WindowUsedQuota: 100,
	}

	snapshot := sub.Snapshot(100)
	if snapshot.TotalRemaining != 100 {
		t.Fatalf("total remaining = %d, want 100", snapshot.TotalRemaining)
	}
	if snapshot.EffectiveResetCap != 200 {
		t.Fatalf("effective reset cap = %d, want 200", snapshot.EffectiveResetCap)
	}
	if snapshot.WindowRemaining != 100 {
		t.Fatalf("window remaining = %d, want 100", snapshot.WindowRemaining)
	}
}
