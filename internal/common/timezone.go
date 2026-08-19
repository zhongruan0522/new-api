package common

import (
	"os"
	"strings"
	"time"
)

var (
	startupLocation     = time.Local
	startupTimezoneName = time.Local.String()
)

func InitStartupTimezone() {
	timezoneName := strings.TrimSpace(os.Getenv("TZ"))
	if timezoneName == "" {
		startupLocation = time.Local
		startupTimezoneName = startupLocation.String()
		return
	}

	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		SysError("failed to load TZ " + timezoneName + ": " + err.Error())
		startupLocation = time.Local
		startupTimezoneName = startupLocation.String()
		return
	}

	time.Local = location
	startupLocation = location
	startupTimezoneName = timezoneName
}

func StartupLocation() *time.Location {
	if startupLocation == nil {
		return time.Local
	}
	return startupLocation
}

func StartupTimezoneName() string {
	if startupTimezoneName == "" {
		return StartupLocation().String()
	}
	return startupTimezoneName
}

func NowInStartupTimezone() time.Time {
	return time.Now().In(StartupLocation())
}
