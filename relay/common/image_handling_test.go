package common

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestApplyImageAutoConvertToURL_SingleUserMessage(t *testing.T) {
	user := dto.Message{Role: "user"}
	user.SetMediaContent([]dto.MediaContent{
		{Type: dto.ContentTypeText, Text: "Describe these images."},
		{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: "https://example.com/a.png", Detail: "high"}},
		{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: "https://example.com/b.png", Detail: "high"}},
	})

	req := &dto.GeneralOpenAIRequest{
		Model:    "deepseek-chat",
		Messages: []dto.Message{user},
	}

	changed, err := ApplyImageAutoConvertToURL(req, nil)
	if err != nil {
		t.Fatalf("expected err=nil, got: %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true, got false")
	}

	got := req.Messages[0].StringContent()
	if !strings.Contains(got, "Describe these images.") {
		t.Fatalf("expected original text to be preserved, got: %q", got)
	}
	if !strings.Contains(got, "https://example.com/a.png") || !strings.Contains(got, "https://example.com/b.png") {
		t.Fatalf("expected both URLs appended, got: %q", got)
	}
	if strings.Count(got, "图片URL：") != 2 {
		t.Fatalf("expected 2 image URL lines, got: %q", got)
	}
	if !strings.Contains(got, "请使用MCP工具查看") {
		t.Fatalf("expected MCP hint appended, got: %q", got)
	}

	// Ensure the message is now text-only.
	for _, part := range req.Messages[0].ParseContent() {
		if part.Type == dto.ContentTypeImageURL {
			t.Fatalf("expected image blocks to be removed, got: %+v", part)
		}
	}
}

func TestApplyImageAutoConvertToURL_AppendsToCorrespondingUserMessage(t *testing.T) {
	user1 := dto.Message{Role: "user"}
	user1.SetMediaContent([]dto.MediaContent{
		{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: "https://example.com/dup.png", Detail: "high"}},
		{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: "https://example.com/dup.png", Detail: "high"}},
	})

	assistant := dto.Message{Role: "assistant"}
	assistant.SetStringContent("OK")

	user2 := dto.Message{Role: "user"}
	user2.SetStringContent("Continue.")

	req := &dto.GeneralOpenAIRequest{
		Model:    "deepseek-chat",
		Messages: []dto.Message{user1, assistant, user2},
	}

	changed, err := ApplyImageAutoConvertToURL(req, nil)
	if err != nil {
		t.Fatalf("expected err=nil, got: %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true, got false")
	}

	// URLs should be appended to the same user message that contained media blocks.
	first := req.Messages[0].StringContent()
	if !req.Messages[0].IsStringContent() {
		t.Fatalf("expected first user message to become string content")
	}
	if strings.Count(first, "https://example.com/dup.png") != 1 {
		t.Fatalf("expected deduped URL appended once to first user, got: %q", first)
	}
	if !strings.Contains(first, "图片URL：https://example.com/dup.png") {
		t.Fatalf("expected image URL hint line, got: %q", first)
	}
	if !strings.Contains(first, "请使用MCP工具查看") {
		t.Fatalf("expected MCP hint appended, got: %q", first)
	}

	// The last user message should stay unchanged (no media -> no appended URLs).
	last := req.Messages[2].StringContent()
	if strings.TrimSpace(last) != "Continue." {
		t.Fatalf("expected last user text preserved, got: %q", last)
	}
	if strings.Contains(last, "dup.png") {
		t.Fatalf("expected no media URL appended to last user, got: %q", last)
	}
}

func TestApplyImageAutoConvertToURL_VideoBlocks(t *testing.T) {
	user := dto.Message{Role: "user"}
	user.SetMediaContent([]dto.MediaContent{
		{Type: dto.ContentTypeText, Text: "Describe this video."},
		{Type: dto.ContentTypeVideoUrl, VideoUrl: &dto.MessageVideoUrl{Url: "https://example.com/a.mp4"}},
	})

	req := &dto.GeneralOpenAIRequest{
		Model:    "deepseek-chat",
		Messages: []dto.Message{user},
	}

	changed, err := ApplyImageAutoConvertToURL(req, nil)
	if err != nil {
		t.Fatalf("expected err=nil, got: %v", err)
	}
	if !changed {
		t.Fatalf("expected changed=true, got false")
	}

	got := req.Messages[0].StringContent()
	if !strings.Contains(got, "Describe this video.") {
		t.Fatalf("expected original text to be preserved, got: %q", got)
	}
	if !strings.Contains(got, "视频URL：https://example.com/a.mp4") {
		t.Fatalf("expected video URL hint line, got: %q", got)
	}
	if !strings.Contains(got, "请使用MCP工具查看") {
		t.Fatalf("expected MCP hint appended, got: %q", got)
	}

	// Ensure the message is now text-only.
	for _, part := range req.Messages[0].ParseContent() {
		if part.Type == dto.ContentTypeVideoUrl {
			t.Fatalf("expected video blocks to be removed, got: %+v", part)
		}
	}
}

