package types

type RelayRequest struct {
	OriginRequest    any
	Format           RelayFormat
	PromptTokenCount int
}

func (r *RelayRequest) CopyOriginRequest() any {
	if r.OriginRequest == nil {
		return nil
	}
	switch v := r.OriginRequest.(type) {
	case *GeneralOpenAIRequest:
		return v.Copy()
	case *GeneralClaudeRequest:
		return v.Copy()
	case *GeneralGeminiRequest:
		return v.Copy()
	case *GeneralRerankRequest:
		return v.Copy()
	case *GeneralEmbeddingRequest:
		return v.Copy()
	default:
		return nil
	}
}
