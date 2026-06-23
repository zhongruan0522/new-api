package common

import (
	"os"
	"path/filepath"
	"time"

	rootcommon "github.com/zhongruan0522/new-api/common"
)

const debugMiniMaxTTSLogPath = "/workspace/new-api/.cursor/debug-fbefe1.log"
const debugMiniMaxTTSSessionID = "fbefe1"

// DebugMiniMaxTTS writes temporary NDJSON logs for the fbefe1 debugging session.
func DebugMiniMaxTTS(location string, message string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	payload := map[string]any{
		"sessionId": debugMiniMaxTTSSessionID,
		"location":  location,
		"message":   message,
		"data":      data,
		"timestamp": time.Now().UnixMilli(),
	}
	line, err := rootcommon.Marshal(payload)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(debugMiniMaxTTSLogPath), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(debugMiniMaxTTSLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}
