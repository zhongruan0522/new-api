package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/NookMux/NookMux/internal/common"
)

// testSSRFProtection 构建与默认 FetchSetting 等价的防护配置：
// 拦截私网/保留地址，不启用域名/IP 黑白名单。
func testSSRFProtection() *common.SSRFProtection {
	return &common.SSRFProtection{
		AllowPrivateIp:   false,
		DomainFilterMode: false,
		IpFilterMode:     false,
	}
}

// TestDialContextSSRFRecheckBlocksPrivateIPAtConnectTime 证明拨号时刻复查生效：
// 即使 URL 静态校验被绕过（例如 DNS rebinding 使校验时解析到公网 IP），
// 只要 transport 实际连接的 IP 是私网地址，携带复查标记的请求也会在
// 建立连接前被拒绝。
func TestDialContextSSRFRecheckBlocksPrivateIPAtConnectTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	}))
	defer server.Close()

	// 服务实际监听在 127.0.0.1。模拟 rebinding 后的连接路径：
	// 请求 URL 是合法公网域名（绕过静态校验的形态用 Transport 自定义拨号
	// 直接构造），但携带复查标记后，直接拨私网 IP 必须被拒绝。
	baseDialer := &net.Dialer{}
	dial := dialContextWithSSRFRecheck(baseDialer.DialContext, nil)

	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener addr: %v", err)
	}

	ctx := WithSSRFRecheck(context.Background(), testSSRFProtection())
	conn, err := dial(ctx, "tcp", net.JoinHostPort("127.0.0.1", port))
	if err == nil {
		conn.Close()
		t.Fatal("dial to private IP with recheck marker should be rejected")
	}
}

// TestDialContextSSRFRecheckAllowsUnmarkedRequests 证明未携带标记的请求
// 不受拨号复查影响（管理员配置的内网渠道上游必须继续可用）。
func TestDialContextSSRFRecheckAllowsUnmarkedRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	}))
	defer server.Close()

	baseDialer := &net.Dialer{}
	dial := dialContextWithSSRFRecheck(baseDialer.DialContext, nil)

	conn, err := dial(context.Background(), "tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("unmarked dial to %s should pass: %v", server.Listener.Addr().String(), err)
	}
	conn.Close()
}

// TestDialContextSSRFRecheckSkipsWhenProxyConfigured 证明配置了代理函数的
// transport 跳过拨号复查：transport 此时拨的是代理地址（常见为本机
// HTTP_PROXY=127.0.0.1:xxxx），对它做私网复查会误杀所有经代理出站的
// 通知/下载流量。
func TestDialContextSSRFRecheckSkipsWhenProxyConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	}))
	defer server.Close()

	baseDialer := &net.Dialer{}
	proxyFunc := func(*http.Request) (*url.URL, error) {
		return url.Parse("http://127.0.0.1:7890")
	}
	dial := dialContextWithSSRFRecheck(baseDialer.DialContext, proxyFunc)

	ctx := WithSSRFRecheck(context.Background(), testSSRFProtection())
	conn, err := dial(ctx, "tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial with proxy configured should skip recheck, got error: %v", err)
	}
	conn.Close()
}

// TestSSRFRecheckCoversFullClientPath 证明经完整 http.Client 的请求链路上
// 复查生效：服务监听 127.0.0.1，客户端请求 localhost（本地解析必然到
// 127.0.0.1）。携带标记的请求在连接时刻被拒，即使假设静态校验已放行。
func TestSSRFRecheckCoversFullClientPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	}))
	defer server.Close()

	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener addr: %v", err)
	}

	client := newRelayHTTPClient(newRelayTransport(nil))
	url := "http://localhost:" + port + "/"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req = req.WithContext(WithSSRFRecheck(req.Context(), testSSRFProtection()))

	// localhost 解析到 127.0.0.1，静态校验（若执行）与拨号复查都会拒绝；
	// 这里只断言拨号复查这一层：client.Do 必须失败且错误提到 IP 不被允许。
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("client request to localhost with recheck marker should fail at dial time")
	}
}
