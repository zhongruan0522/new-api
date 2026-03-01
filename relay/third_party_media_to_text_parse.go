package relay

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
)

func parseThirdPartyOutputText(apiType int, body []byte) (string, error) {
	switch apiType {
	case constant.APITypeAnthropic:
		return parseClaudeOutputText(body)
	case constant.APITypeGemini:
		return parseGeminiOutputText(body)
	default:
		return parseOpenAIOutputText(body)
	}
}

func parseOpenAIOutputText(body []byte) (string, error) {
	var resp dto.OpenAITextResponse
	if err := common.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if oaiErr := resp.GetOpenAIError(); oaiErr != nil {
		return "", fmt.Errorf("openai_error: %s", strings.TrimSpace(oaiErr.Message))
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("empty choices")
	}
	text := strings.TrimSpace(resp.Choices[0].Message.StringContent())
	if text == "" {
		return "", errors.New("empty content")
	}
	return text, nil
}

func parseClaudeOutputText(body []byte) (string, error) {
	var resp dto.ClaudeResponse
	if err := common.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if claudeErr := resp.GetClaudeError(); claudeErr != nil {
		return "", fmt.Errorf("claude_error: %s", strings.TrimSpace(claudeErr.Message))
	}
	if len(resp.Content) == 0 {
		return "", errors.New("empty content")
	}
	var b strings.Builder
	for _, part := range resp.Content {
		if part.Type != "text" {
			continue
		}
		b.WriteString(part.GetText())
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", errors.New("empty content")
	}
	return text, nil
}

func parseGeminiOutputText(body []byte) (string, error) {
	var resp dto.GeminiChatResponse
	if err := common.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if len(resp.Candidates) == 0 {
		return "", errors.New("empty candidates")
	}

	var b strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if strings.TrimSpace(part.Text) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(part.Text)
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", errors.New("empty content")
	}
	return text, nil
}
