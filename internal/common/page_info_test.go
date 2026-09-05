package common

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetPageQueryClampsNegativeValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		query        string
		wantPage     int
		wantPageSize int
	}{
		{name: "negative page size", query: "/?p=1&page_size=-1", wantPage: 1, wantPageSize: ItemsPerPage},
		{name: "negative page", query: "/?p=-1&page_size=20", wantPage: 1, wantPageSize: 20},
		{name: "negative alias page size", query: "/?p=1&ps=-1", wantPage: 1, wantPageSize: ItemsPerPage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", tt.query, nil)

			pageInfo := GetPageQuery(c)
			if pageInfo.GetPage() != tt.wantPage {
				t.Fatalf("page = %d, want %d", pageInfo.GetPage(), tt.wantPage)
			}
			if pageInfo.GetPageSize() != tt.wantPageSize {
				t.Fatalf("page size = %d, want %d", pageInfo.GetPageSize(), tt.wantPageSize)
			}
			if pageInfo.GetStartIdx() < 0 {
				t.Fatalf("start index = %d, want non-negative", pageInfo.GetStartIdx())
			}
		})
	}
}
