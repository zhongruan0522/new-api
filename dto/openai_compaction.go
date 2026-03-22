package dto

import (
	"encoding/json"

	"github.com/zhongruan0522/new-api/types"
)

type OpenAIResponsesCompactionResponse struct {
	ID        string          `json:"id"`
	Object    string          `json:"object"`
	CreatedAt int             `json:"created_at"`
	Output    json.RawMessage `json:"output"`
	Usage     *Usage          `json:"usage"`
	Error     any             `json:"error,omitempty"`
}

func (o *OpenAIResponsesCompactionResponse) GetOpenAIError() *types.OpenAIError {
	return GetOpenAIError(o.Error)
}
