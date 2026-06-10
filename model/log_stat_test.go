package model

import (
	"testing"
	"time"
)

func TestSumUsedQuotaAppliesLogOnlyFilters(t *testing.T) {
	setupLogAdminInfoTestDB(t)

	now := time.Now().Unix()
	logs := []*Log{
		{
			UserId:            1,
			CreatedAt:         now,
			Type:              LogTypeConsume,
			Username:          "target-user",
			TokenName:         "target-token",
			ModelName:         "gpt-target",
			Quota:             120,
			PromptTokens:      10,
			CompletionTokens:  20,
			ChannelId:         1,
			Group:             "default",
			Ip:                "203.0.113.10",
			RequestId:         "target-request",
			UpstreamRequestId: "target-upstream",
			Other:             `{"ua":"TargetAgent/1.0","x_title":"Target Title","http_referer":"https://target.example/path"}`,
		},
		{
			UserId:            2,
			CreatedAt:         now,
			Type:              LogTypeConsume,
			Username:          "other-user",
			TokenName:         "other-token",
			ModelName:         "gpt-other",
			Quota:             300,
			PromptTokens:      100,
			CompletionTokens:  200,
			ChannelId:         2,
			Group:             "vip",
			Ip:                "198.51.100.20",
			RequestId:         "other-request",
			UpstreamRequestId: "other-upstream",
			Other:             `{"ua":"OtherAgent/1.0","x_title":"Other Title","http_referer":"https://other.example/path"}`,
		},
	}
	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatalf("create logs: %v", err)
	}

	var matched int64
	if err := LOG_DB.Model(&Log{}).Where("request_id = ?", "target-request").Count(&matched).Error; err != nil {
		t.Fatalf("count target log: %v", err)
	}
	if matched != 1 {
		t.Fatalf("target log count = %d, want 1", matched)
	}

	stat, err := SumUsedQuota(LogTypeConsume, now-1, now+1, LogStatFilter{
		RequestId:         "target-request",
		UpstreamRequestId: "target-upstream",
		Ip:                "203.0.113",
		Ua:                "TargetAgent",
		XTitle:            "Target Title",
		HttpReferer:       "target.example",
	})
	if err != nil {
		t.Fatalf("SumUsedQuota error = %v", err)
	}

	if stat.Quota != 120 {
		t.Fatalf("quota = %d, want 120", stat.Quota)
	}
	if stat.Rpm != 1 {
		t.Fatalf("rpm = %d, want 1", stat.Rpm)
	}
	if stat.Tpm != 30 {
		t.Fatalf("tpm = %d, want 30", stat.Tpm)
	}
	if stat.SuccessCount != 1 {
		t.Fatalf("success_count = %d, want 1", stat.SuccessCount)
	}
}
