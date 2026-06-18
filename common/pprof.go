package common

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"time"

	"github.com/shirou/gopsutil/cpu"

	// Trigger pprofinit.init which registers net/http/pprof handlers.
	"github.com/zhongruan0522/new-api/common/pprofinit"
)

// EnablePprofServer imports net/http/pprof (which registers handlers on
// http.DefaultServeMux via init()) and starts the debug HTTP server on :8005.
//
// This is called only when ENABLE_PPROF=true so that pprof's init overhead is
// avoided in normal production builds. The blank import of net/http/pprof is
// in the pprofinit package; referencing it here forces the compiler to link
// those handlers.
func EnablePprofServer() {
	// Force the pprofinit package init to run (it registers handlers).
	_ = pprofinit.PackageInitMarker
	registerPprofHandlers()
	log.Println(http.ListenAndServe("0.0.0.0:8005", http.DefaultServeMux))
}

// registerPprofHandlers is a no-op placeholder; the actual registration happens
// via pprofinit's init(). Keeping this function makes the intent explicit and
// gives a place to add manual handler wiring if needed in the future.
func registerPprofHandlers() {}

// Monitor 定时监控cpu使用率，超过阈值输出pprof文件
func Monitor() {
	for {
		percent, err := cpu.Percent(time.Second, false)
		if err != nil {
			panic(err)
		}
		if percent[0] > 80 {
			fmt.Println("cpu usage too high")
			// write pprof file
			if _, err := os.Stat("./pprof"); os.IsNotExist(err) {
				err := os.Mkdir("./pprof", os.ModePerm)
				if err != nil {
					SysLog("创建pprof文件夹失败 " + err.Error())
					continue
				}
			}
			f, err := os.Create("./pprof/" + fmt.Sprintf("cpu-%s.pprof", time.Now().Format("20060102150405")))
			if err != nil {
				SysLog("创建pprof文件失败 " + err.Error())
				continue
			}
			err = pprof.StartCPUProfile(f)
			if err != nil {
				SysLog("启动pprof失败 " + err.Error())
				continue
			}
			time.Sleep(10 * time.Second) // profile for 30 seconds
			pprof.StopCPUProfile()
			f.Close()
		}
		time.Sleep(30 * time.Second)
	}
}
