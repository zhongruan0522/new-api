package common

import (
	"testing"
	"time"
)

func TestInitStartupTimezoneCapturesTZ(t *testing.T) {
	previousLocation := startupLocation
	previousName := startupTimezoneName
	previousTimeLocal := time.Local
	t.Cleanup(func() {
		startupLocation = previousLocation
		startupTimezoneName = previousName
		time.Local = previousTimeLocal
	})

	t.Setenv("TZ", "Asia/Shanghai")
	InitStartupTimezone()

	if StartupTimezoneName() != "Asia/Shanghai" {
		t.Fatalf("StartupTimezoneName() = %q, want Asia/Shanghai", StartupTimezoneName())
	}
	if StartupLocation().String() != "Asia/Shanghai" {
		t.Fatalf("StartupLocation() = %q, want Asia/Shanghai", StartupLocation().String())
	}
	if NowInStartupTimezone().Location().String() != "Asia/Shanghai" {
		t.Fatalf("NowInStartupTimezone location = %q, want Asia/Shanghai", NowInStartupTimezone().Location().String())
	}
}
