package performance

import (
	"testing"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/stretchr/testify/require"
)

func TestDefaultPerformanceSettingUsesHigherDiskThreshold(t *testing.T) {
	setting := GetPerformanceSetting()
	require.Equal(t, 95, setting.MonitorDiskThreshold)

	monitorConfig := common.GetPerformanceMonitorConfig()
	require.Equal(t, 95, monitorConfig.DiskThreshold)
}
