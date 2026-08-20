package misccontroller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/gin-gonic/gin"
)

func runGetStatusUserModules(t *testing.T, role int) map[string]json.RawMessage {
	t.Helper()
	gin.SetMode(gin.TestMode)

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	old, existed := common.OptionMap["SidebarModulesAdmin"]
	common.OptionMap["SidebarModulesAdmin"] = `{"chat":{"enabled":true,"playground":false},"console":{"enabled":true,"log":false},"admin":{"enabled":true,"channel":false}}`
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		if existed {
			common.OptionMap["SidebarModulesAdmin"] = old
		} else {
			delete(common.OptionMap, "SidebarModulesAdmin")
		}
		common.OptionMapRWMutex.Unlock()
	})

	resp := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(resp)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/status/user_modules", nil)
	c.Set("role", role)
	GetStatusUserModules(c)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var body struct {
		Data struct {
			SidebarModulesAdmin string `json:"SidebarModulesAdmin"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	sections := make(map[string]json.RawMessage)
	if err := json.Unmarshal([]byte(body.Data.SidebarModulesAdmin), &sections); err != nil {
		t.Fatalf("unmarshal sidebar config %q: %v", body.Data.SidebarModulesAdmin, err)
	}
	return sections
}

// 非管理员调用 user_modules：面向用户的段（chat/console）正常下发，
// admin 段必须被剥离，保证管理面能力结构不向普通用户暴露。
func TestGetStatusUserModulesStripsAdminSectionForCommonUser(t *testing.T) {
	sections := runGetStatusUserModules(t, common.RoleCommonUser)

	if _, ok := sections["admin"]; ok {
		t.Fatal("user_modules must strip the admin section for non-admin callers")
	}
	if _, ok := sections["chat"]; !ok {
		t.Fatal("user_modules must keep user-facing chat section")
	}
	if _, ok := sections["console"]; !ok {
		t.Fatal("user_modules must keep user-facing console section")
	}
}

// 管理员调用 user_modules：返回全量配置（含 admin 段）。
func TestGetStatusUserModulesKeepsAdminSectionForAdmin(t *testing.T) {
	sections := runGetStatusUserModules(t, common.RoleAdminUser)

	if _, ok := sections["admin"]; !ok {
		t.Fatal("user_modules must keep the admin section for admin callers")
	}
	if _, ok := sections["chat"]; !ok {
		t.Fatal("user_modules must keep chat section for admin callers")
	}
}

// 空配置/非法 JSON 不应报错：原样透传，由前端回落默认配置。
func TestStripAdminSidebarSectionPassthrough(t *testing.T) {
	for _, raw := range []string{"", "   ", "not-json"} {
		if out, ok := stripAdminSidebarSection(raw); ok {
			t.Fatalf("stripAdminSidebarSection(%q) should fail passthrough, got %q", raw, out)
		}
	}
	out, ok := stripAdminSidebarSection(`{"chat":{"enabled":true}}`)
	if !ok || out != `{"chat":{"enabled":true}}` {
		t.Fatalf("config without admin section should return as-is, got %q ok=%v", out, ok)
	}
	out, ok = stripAdminSidebarSection(`{"chat":{"enabled":true},"admin":{"enabled":true}}`)
	if !ok {
		t.Fatal("strip should succeed")
	}
	parsed := make(map[string]json.RawMessage)
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("stripped output must be valid JSON: %v", err)
	}
	if _, exists := parsed["admin"]; exists {
		t.Fatal("admin section must be removed")
	}
	if _, exists := parsed["chat"]; !exists {
		t.Fatal("chat section must survive stripping")
	}
}
