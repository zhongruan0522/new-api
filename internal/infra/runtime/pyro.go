package runtime

import (
	"runtime"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/grafana/pyroscope-go"
)

func StartPyroScope() error {

	pyroscopeUrl := common.GetEnvOrDefaultString("PYROSCOPE_URL", "")
	if pyroscopeUrl == "" {
		return nil
	}

	pyroscopeAppName := common.GetEnvOrDefaultString("PYROSCOPE_APP_NAME", "nookmux")
	pyroscopeBasicAuthUser := common.GetEnvOrDefaultString("PYROSCOPE_BASIC_AUTH_USER", "")
	pyroscopeBasicAuthPassword := common.GetEnvOrDefaultString("PYROSCOPE_BASIC_AUTH_PASSWORD", "")
	pyroscopeHostname := common.GetEnvOrDefaultString("HOSTNAME", "nookmux")

	mutexRate := common.GetEnvOrDefault("PYROSCOPE_MUTEX_RATE", 5)
	blockRate := common.GetEnvOrDefault("PYROSCOPE_BLOCK_RATE", 5)

	runtime.SetMutexProfileFraction(mutexRate)
	runtime.SetBlockProfileRate(blockRate)

	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: pyroscopeAppName,

		ServerAddress:     pyroscopeUrl,
		BasicAuthUser:     pyroscopeBasicAuthUser,
		BasicAuthPassword: pyroscopeBasicAuthPassword,

		Logger: nil,

		Tags: map[string]string{"hostname": pyroscopeHostname},

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
