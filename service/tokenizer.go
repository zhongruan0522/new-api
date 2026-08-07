package service

import (
	"strings"
	"sync"

	"github.com/tiktoken-go/tokenizer"
	"github.com/tiktoken-go/tokenizer/codec"
	"github.com/NookMux/NookMux/common"
)

const maxTokenEncoderCacheSize = 64

var defaultTokenEncoder tokenizer.Codec

// tokenEncoderMap is used to store token encoders for different models
var tokenEncoderMap = make(map[string]tokenizer.Codec)
var tokenEncoderOrder []string

// tokenEncoderMutex protects tokenEncoderMap for concurrent access
var tokenEncoderMutex sync.RWMutex

func InitTokenEncoders() {
	common.SysLog("token encoders will be initialized on first use")
}

func getDefaultTokenEncoder() tokenizer.Codec {
	tokenEncoderMutex.RLock()
	encoder := defaultTokenEncoder
	tokenEncoderMutex.RUnlock()
	if encoder != nil {
		return encoder
	}

	tokenEncoderMutex.Lock()
	defer tokenEncoderMutex.Unlock()
	return getDefaultTokenEncoderLocked()
}

func getDefaultTokenEncoderLocked() tokenizer.Codec {
	if defaultTokenEncoder == nil {
		defaultTokenEncoder = codec.NewCl100kBase()
	}
	return defaultTokenEncoder
}

func getTokenEncoder(model string) tokenizer.Codec {
	model = normalizeTokenEncoderModel(model)
	if model == "" {
		return getDefaultTokenEncoder()
	}

	// First, try to get the encoder from cache with read lock
	tokenEncoderMutex.RLock()
	if encoder, exists := tokenEncoderMap[model]; exists {
		tokenEncoderMutex.RUnlock()
		return encoder
	}
	tokenEncoderMutex.RUnlock()

	// If not in cache, create new encoder with write lock
	tokenEncoderMutex.Lock()
	defer tokenEncoderMutex.Unlock()

	// Double-check if another goroutine already created the encoder
	if encoder, exists := tokenEncoderMap[model]; exists {
		return encoder
	}

	// Create new encoder
	modelCodec, err := tokenizer.ForModel(tokenizer.Model(model))
	if err != nil {
		return getDefaultTokenEncoderLocked()
	}

	// Cache the new encoder
	storeTokenEncoderLocked(model, modelCodec)
	return modelCodec
}

func normalizeTokenEncoderModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(model, "gpt-5"):
		return "gpt-4o"
	case strings.HasPrefix(model, "gpt-4o"):
		return "gpt-4o"
	case strings.HasPrefix(model, "gpt-4.1"):
		return "gpt-4"
	case strings.HasPrefix(model, "gpt-4"):
		return "gpt-4"
	case strings.HasPrefix(model, "gpt-3.5"):
		return "gpt-3.5-turbo"
	case strings.HasPrefix(model, "o1"):
		return "o1"
	case strings.HasPrefix(model, "o3"):
		return "o3"
	case strings.HasPrefix(model, "o4"):
		return "o4-mini"
	case strings.HasPrefix(model, "chatgpt"):
		return "gpt-4o"
	default:
		return model
	}
}

func storeTokenEncoderLocked(model string, encoder tokenizer.Codec) {
	if len(tokenEncoderMap) >= maxTokenEncoderCacheSize {
		for len(tokenEncoderOrder) > 0 {
			evict := tokenEncoderOrder[0]
			tokenEncoderOrder = tokenEncoderOrder[1:]
			if _, ok := tokenEncoderMap[evict]; ok {
				delete(tokenEncoderMap, evict)
				break
			}
		}
	}
	tokenEncoderMap[model] = encoder
	tokenEncoderOrder = append(tokenEncoderOrder, model)
}

func getTokenNum(tokenEncoder tokenizer.Codec, text string) int {
	if text == "" {
		return 0
	}
	tkm, _ := tokenEncoder.Count(text)
	return tkm
}
