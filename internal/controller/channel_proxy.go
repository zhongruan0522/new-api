package controller

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/i18n"
	"github.com/NookMux/NookMux/pkg/jsonx"

	httpclient "github.com/NookMux/NookMux/internal/infra/httpclient"
	"github.com/gin-gonic/gin"
)

// proxyTestEndpoint is the public IP echo service used to verify that the
// proxy actually routes outbound traffic. It responds with the caller's IP
// address as plain text.
const proxyTestEndpoint = "https://api.ipify.org"

// proxyTestTimeout caps how long a proxy connectivity probe may take so a
// misconfigured/dead proxy cannot keep the admin UI blocked.
const proxyTestTimeout = 15 * time.Second

// ProxyTestStatus describes the outcome of a proxy connectivity probe.
//
//   - "success": the proxy works and the exit IP was resolved
//   - "invalid": the proxy URL is empty or uses an unsupported scheme
//   - "failed":  the URL is well-formed but the proxy could not be reached
type ProxyTestStatus string

const (
	ProxyTestStatusSuccess ProxyTestStatus = "success"
	ProxyTestStatusInvalid ProxyTestStatus = "invalid"
	ProxyTestStatusFailed  ProxyTestStatus = "failed"
)

// proxyTestRequest is the JSON body accepted by TestProxy.
type proxyTestRequest struct {
	Proxy string `json:"proxy"`
}

// TestProxy validates a proxy URL and attempts an outbound request through it
// to resolve the exit IP. It is intended for the channel editor's "test proxy"
// affordance.
func TestProxy(c *gin.Context) {
	var req proxyTestRequest
	if err := jsonx.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidRequestBody)
		return
	}

	proxyURL := strings.TrimSpace(req.Proxy)
	if proxyURL == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"status":  string(ProxyTestStatusInvalid),
				"message": i18n.T(c, i18n.MsgChannelProxyEmpty),
			},
		})
		return
	}

	// NewProxyHttpClient parses and validates the scheme (http, https, socks5,
	// socks5h) before attempting any connection, so a returned error means the
	// URL itself is non-compliant rather than a connectivity problem.
	client, err := httpclient.NewProxyHttpClient(proxyURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"status":  string(ProxyTestStatusInvalid),
				"message": i18n.T(c, i18n.MsgChannelProxyInvalid, map[string]any{"Error": err.Error()}),
			},
		})
		return
	}

	// Use a dedicated client so the cached proxy client's timeout is not
	// mutated for concurrent relay traffic.
	testClient := &http.Client{
		Transport:     client.Transport,
		Timeout:       proxyTestTimeout,
		CheckRedirect: client.CheckRedirect,
	}

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, proxyTestEndpoint, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"status":  string(ProxyTestStatusFailed),
				"message": i18n.T(c, i18n.MsgChannelProxyRequestFailed, map[string]any{"Error": err.Error()}),
			},
		})
		return
	}

	resp, err := testClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"status":  string(ProxyTestStatusFailed),
				"message": i18n.T(c, i18n.MsgChannelProxyRequestFailed, map[string]any{"Error": err.Error()}),
			},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"status":  string(ProxyTestStatusFailed),
				"message": i18n.T(c, i18n.MsgChannelProxyUnexpectedStatus, map[string]any{"Status": fmt.Sprintf("%d", resp.StatusCode)}),
			},
		})
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"status":  string(ProxyTestStatusFailed),
				"message": i18n.T(c, i18n.MsgChannelProxyRequestFailed, map[string]any{"Error": err.Error()}),
			},
		})
		return
	}

	exitIP := strings.TrimSpace(string(body))
	if exitIP == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"status":  string(ProxyTestStatusFailed),
				"message": i18n.T(c, i18n.MsgChannelProxyEmptyResponse),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"status": string(ProxyTestStatusSuccess),
			"ip":     exitIP,
		},
	})
}
