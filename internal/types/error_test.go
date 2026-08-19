package types

import (
	"errors"
	"strings"
	"testing"
)

func TestSetExemptStrings_PreservesModelAndGroupInMaskSensitiveError(t *testing.T) {
	e := NewError(
		errors.New("分组 minimax 下模型 speech-02-hd 的可用渠道不存在（retry）"),
		ErrorCodeGetChannelFailed,
	)
	e.SetExemptStrings("minimax", "speech-02-hd")

	got := e.MaskSensitiveError()
	if strings.Contains(got, "*minimax") {
		t.Errorf("group name 'minimax' should be exempt, got: %q", got)
	}
	if strings.Contains(got, "*speech-02-hd") {
		t.Errorf("model name 'speech-02-hd' should be exempt, got: %q", got)
	}
	if !strings.Contains(got, "minimax") || !strings.Contains(got, "speech-02-hd") {
		t.Errorf("group and model names should be preserved, got: %q", got)
	}
}

func TestSetExemptStrings_MasksNonExemptBrandInToOpenAIError(t *testing.T) {
	e := NewOpenAIError(
		errors.New("upstream minimax service unavailable, speech-02-turbo model error"),
		ErrorCodeBadResponseStatusCode,
		502,
	)
	e.SetExemptStrings("speech-02-hd")

	msg := e.ToOpenAIError().Message
	if strings.Contains(msg, "minimax") {
		t.Errorf("non-exempt 'minimax' should be masked, got: %q", msg)
	}
	if strings.Contains(msg, "speech-02-turbo") {
		t.Errorf("non-exempt 'speech-02-turbo' prefix should be masked, got: %q", msg)
	}
}

func TestSetExemptStrings_NoExemptions_MasksEverything(t *testing.T) {
	e := NewError(
		errors.New("minimax speech-02-hd error"),
		ErrorCodeBadResponse,
	)
	// No SetExemptStrings call

	got := e.MaskSensitiveError()
	if strings.Contains(got, "minimax") {
		t.Errorf("'minimax' should be masked without exemptions, got: %q", got)
	}
	if strings.Contains(got, "speech-") {
		t.Errorf("'speech-' should be masked without exemptions, got: %q", got)
	}
}
