package performance_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/NookMux/NookMux/common"
)

func TestDefaultPerformanceSettingUsesHigherDiskThreshold(t *testing.T) {
	setting := GetPerformanceSetting()
	require.Equal(t, 95, setting.MonitorDiskThreshold)

	monitorConfig := common.GetPerformanceMonitorConfig()
	require.Equal(t, 95, monitorConfig.DiskThreshold)
}
