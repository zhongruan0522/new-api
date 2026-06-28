package common

import (
	"fmt"
	"net/http"
	"os"
	runtimepprof "runtime/pprof"
	"time"

	"github.com/shirou/gopsutil/cpu"

	// Register pprof debug handlers on http.DefaultServeMux via init().
	// The handlers are only reachable through the debug server started by
	// EnablePprofServer when ENABLE_PPROF=true.
	_ "net/http/pprof"
)

// EnablePprofServer starts the debug HTTP server on :8005 with pprof handlers
// registered on http.DefaultServeMux.
//
// The net/http/pprof blank import at the package level registers the handlers
// unconditionally, but the server itself only starts when ENABLE_PPROF=true.
func EnablePprofServer() {
	if err := http.ListenAndServe("0.0.0.0:8005", http.DefaultServeMux); err != nil {
		SysError(fmt.Sprintf("pprof server stopped: %v", err))
	}
}

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
			err = runtimepprof.StartCPUProfile(f)
			if err != nil {
				SysLog("启动pprof失败 " + err.Error())
				continue
			}
			time.Sleep(10 * time.Second) // profile for 30 seconds
			runtimepprof.StopCPUProfile()
			f.Close()
		}
		time.Sleep(30 * time.Second)
	}
}
