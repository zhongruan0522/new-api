package relay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhongruan0522/new-api/common"

	"github.com/gin-gonic/gin"
)

func newStoredAssetSignatureContext(rawURL string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, rawURL, nil)
	return c, recorder
}

func TestVerifyStoredAssetSignatureRejectsPermanentSignature(t *testing.T) {
	id := "asset-1"
	sig := common.GenerateHMAC(fmt.Sprintf("%s:%s", "stored_video", id))
	c, recorder := newStoredAssetSignatureContext("/mcp/video/" + id + "?sig=" + sig)

	_, _, ok := verifyStoredAssetSignature(c, "stored_video", id)
	if ok {
		t.Fatalf("verifyStoredAssetSignature accepted signature without exp")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestVerifyStoredAssetSignatureRejectsExpiredSignature(t *testing.T) {
	id := "asset-2"
	exp := time.Now().Add(-time.Minute).Unix()
	sig := common.GenerateHMAC(fmt.Sprintf("%s:%s:%d", "stored_image", id, exp))
	c, recorder := newStoredAssetSignatureContext(fmt.Sprintf("/mcp/image/%s?exp=%d&sig=%s", id, exp, sig))

	_, _, ok := verifyStoredAssetSignature(c, "stored_image", id)
	if ok {
		t.Fatalf("verifyStoredAssetSignature accepted expired signature")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestVerifyStoredAssetSignatureAcceptsTimeBoundSignature(t *testing.T) {
	id := "asset-3"
	exp := time.Now().Add(time.Hour).Unix()
	sig := common.GenerateHMAC(fmt.Sprintf("%s:%s:%d", "stored_video", id, exp))
	c, recorder := newStoredAssetSignatureContext(fmt.Sprintf("/mcp/video/%s?exp=%d&sig=%s", id, exp, sig))

	gotExp, _, ok := verifyStoredAssetSignature(c, "stored_video", id)
	if !ok {
		t.Fatalf("verifyStoredAssetSignature rejected valid signature, body: %s", recorder.Body.String())
	}
	if gotExp != exp {
		t.Fatalf("exp = %d, want %d", gotExp, exp)
	}
}
