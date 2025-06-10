package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"one-api/common"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type UptimeKumaMonitor struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type UptimeKumaGroup struct {
	ID          int                  `json:"id"`
	Name        string               `json:"name"`
	Weight      int                  `json:"weight"`
	MonitorList []UptimeKumaMonitor  `json:"monitorList"`
}

type UptimeKumaHeartbeat struct {
	Status int      `json:"status"`
	Time   string   `json:"time"`
	Msg    string   `json:"msg"`
	Ping   *float64 `json:"ping"`
}

type UptimeKumaStatusResponse struct {
	PublicGroupList []UptimeKumaGroup `json:"publicGroupList"`
}

type UptimeKumaHeartbeatResponse struct {
	HeartbeatList map[string][]UptimeKumaHeartbeat `json:"heartbeatList"`
	UptimeList    map[string]float64               `json:"uptimeList"`
}

type MonitorStatus struct {
	Name   string  `json:"name"`
	Uptime float64 `json:"uptime"`
	Status int     `json:"status"`
}

var (
	ErrUpstreamNon200 = errors.New("upstream non-200")
	ErrTimeout        = errors.New("context deadline exceeded")
)

func getAndDecode(ctx context.Context, client *http.Client, url string, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return ErrTimeout
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrUpstreamNon200
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

func GetUptimeKumaStatus(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	uptimeKumaUrl := common.OptionMap["UptimeKumaUrl"]
	slug := common.OptionMap["UptimeKumaSlug"]
	common.OptionMapRWMutex.RUnlock()

	if uptimeKumaUrl == "" || slug == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data":    []MonitorStatus{},
		})
		return
	}

	uptimeKumaUrl = strings.TrimSuffix(uptimeKumaUrl, "/")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	client := &http.Client{}

	statusPageUrl := fmt.Sprintf("%s/api/status-page/%s", uptimeKumaUrl, slug)
	heartbeatUrl := fmt.Sprintf("%s/api/status-page/heartbeat/%s", uptimeKumaUrl, slug)

	var (
		statusData    UptimeKumaStatusResponse
		heartbeatData UptimeKumaHeartbeatResponse
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return getAndDecode(gCtx, client, statusPageUrl, &statusData)
	})

	g.Go(func() error {
		return getAndDecode(gCtx, client, heartbeatUrl, &heartbeatData)
	})

	if err := g.Wait(); err != nil {
		switch err {
		case ErrUpstreamNon200:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "上游接口出现问题",
			})
		case ErrTimeout:
			c.JSON(http.StatusRequestTimeout, gin.H{
				"success": false,
				"message": "请求上游接口超时",
			})
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": err.Error(),
			})
		}
		return
	}

	var monitors []MonitorStatus
	for _, group := range statusData.PublicGroupList {
		for _, monitor := range group.MonitorList {
			monitorStatus := MonitorStatus{
				Name:   monitor.Name,
				Uptime: 0.0,
				Status: 0,
			}

			uptimeKey := fmt.Sprintf("%d_24", monitor.ID)
			if uptime, exists := heartbeatData.UptimeList[uptimeKey]; exists {
				monitorStatus.Uptime = uptime
			}

			heartbeatKey := fmt.Sprintf("%d", monitor.ID)
			if heartbeats, exists := heartbeatData.HeartbeatList[heartbeatKey]; exists && len(heartbeats) > 0 {
				latestHeartbeat := heartbeats[0]
				monitorStatus.Status = latestHeartbeat.Status
			}

			monitors = append(monitors, monitorStatus)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    monitors,
	})
} 