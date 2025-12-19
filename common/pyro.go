package common

import (
	"runtime"

	"github.com/grafana/pyroscope-go"
)

func StartPyroScope() error {

	pyroscopeUrl := GetEnvOrDefaultString("PYROSCOPE_URL", "")
	if pyroscopeUrl == "" {
		return nil
	}

	pyroscopeAppName := GetEnvOrDefaultString("PYROSCOPE_APP_NAME", "new-api")

	// These 2 lines are only required if you're using mutex or block profiling
	// Read the explanation below for how to set these rates:
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: pyroscopeAppName,

		ServerAddress: pyroscopeUrl,

		Logger: nil,

		Tags: map[string]string{"hostname": GetEnvOrDefaultString("HOSTNAME", "new-api")},

		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return err
	}
	return nil
}
