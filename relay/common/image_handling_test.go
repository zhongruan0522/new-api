package common

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestApplyImageAutoConvertToURL_SingleUserMessage(t *testing.T) {
	user := dto.Message{Role: "user"}
	user.SetMediaContent([]dto.MediaContent{
		{Type: dto.ContentTypeText, Text: "请描述这张图"},
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
	if !strings.Contains(got, "请描述这张图") {
		t.Fatalf("expected original text to be preserved, got: %q", got)
	}
	if !strings.Contains(got, "Media URLs:") {
		t.Fatalf("expected Media URLs marker, got: %q", got)
	}
	if !strings.Contains(got, "https://example.com/a.png") || !strings.Contains(got, "https://example.com/b.png") {
		t.Fatalf("expected both URLs appended, got: %q", got)
	}

	// Ensure the message is now text-only.
	for _, part := range req.Messages[0].ParseContent() {
		if part.Type == dto.ContentTypeImageURL {
			t.Fatalf("expected image blocks to be removed, got: %+v", part)
		}
	}
}

func TestApplyImageAutoConvertToURL_AppendsToLastUser(t *testing.T) {
	user1 := dto.Message{Role: "user"}
	user1.SetMediaContent([]dto.MediaContent{
		{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: "https://example.com/dup.png", Detail: "high"}},
		{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: "https://example.com/dup.png", Detail: "high"}},
	})

	assistant := dto.Message{Role: "assistant"}
	assistant.SetStringContent("好的")

	user2 := dto.Message{Role: "user"}
	user2.SetStringContent("继续")

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

	// First user message should become text-only, even if it had only images.
	if !req.Messages[0].IsStringContent() {
		t.Fatalf("expected first user message to become string content")
	}
	if strings.TrimSpace(req.Messages[0].StringContent()) == "" {
		t.Fatalf("expected placeholder text for image-only user message")
	}
	if strings.TrimSpace(req.Messages[0].StringContent()) != "[media]" {
		t.Fatalf("expected [media] placeholder, got: %q", req.Messages[0].StringContent())
	}

	// URLs should be appended to the last user message.
	last := req.Messages[2].StringContent()
	if !strings.Contains(last, "继续") {
		t.Fatalf("expected last user text preserved, got: %q", last)
	}
	if strings.Count(last, "https://example.com/dup.png") != 1 {
		t.Fatalf("expected deduped URL appended once, got: %q", last)
	}
}

func TestApplyImageAutoConvertToURL_VideoBlocks(t *testing.T) {
	user := dto.Message{Role: "user"}
	user.SetMediaContent([]dto.MediaContent{
		{Type: dto.ContentTypeText, Text: "请描述这个视频"},
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
	if !strings.Contains(got, "请描述这个视频") {
		t.Fatalf("expected original text to be preserved, got: %q", got)
	}
	if !strings.Contains(got, "Media URLs:") {
		t.Fatalf("expected Media URLs marker, got: %q", got)
	}
	if !strings.Contains(got, "[video]") || !strings.Contains(got, "https://example.com/a.mp4") {
		t.Fatalf("expected video URL appended with label, got: %q", got)
	}
}
