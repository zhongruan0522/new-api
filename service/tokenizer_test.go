package service

import (
	"fmt"
	"testing"

	"github.com/tiktoken-go/tokenizer"
	"github.com/tiktoken-go/tokenizer/codec"
)

func resetTokenEncoderCacheForTest(t *testing.T) {
	t.Helper()

	restore := resetTokenEncoderCache()
	t.Cleanup(restore)
}

func resetTokenEncoderCache() func() {
	tokenEncoderMutex.Lock()
	oldDefault := defaultTokenEncoder
	oldMap := tokenEncoderMap
	oldOrder := tokenEncoderOrder
	defaultTokenEncoder = codec.NewCl100kBase()
	tokenEncoderMap = make(map[string]tokenizer.Codec)
	tokenEncoderOrder = nil
	tokenEncoderMutex.Unlock()

	return func() {
		tokenEncoderMutex.Lock()
		defaultTokenEncoder = oldDefault
		tokenEncoderMap = oldMap
		tokenEncoderOrder = oldOrder
		tokenEncoderMutex.Unlock()
	}
}

func TestNormalizeTokenEncoderModelCollapsesDynamicVariants(t *testing.T) {
	tests := map[string]string{
		" GPT-4o-2024-08-06 ":   "gpt-4o",
		"gpt-5-mini-2025-08-07": "gpt-4o",
		"gpt-4.1-mini":          "gpt-4",
		"gpt-3.5-turbo-0125":    "gpt-3.5-turbo",
		"chatgpt-4o-latest":     "gpt-4o",
		"o3-mini":               "o3",
	}

	for input, want := range tests {
		if got := normalizeTokenEncoderModel(input); got != want {
			t.Fatalf("normalizeTokenEncoderModel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestUnknownTokenEncoderModelsDoNotGrowCache(t *testing.T) {
	resetTokenEncoderCacheForTest(t)

	for i := 0; i < maxTokenEncoderCacheSize*2; i++ {
		_ = getTokenEncoder(fmt.Sprintf("unknown-dynamic-model-%d", i))
	}

	tokenEncoderMutex.RLock()
	cacheSize := len(tokenEncoderMap)
	tokenEncoderMutex.RUnlock()

	if cacheSize != 0 {
		t.Fatalf("unknown model cache size = %d, want 0", cacheSize)
	}
}

func BenchmarkUnknownTokenEncoderModels(b *testing.B) {
	restore := resetTokenEncoderCache()
	defer restore()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = getTokenEncoder(fmt.Sprintf("unknown-dynamic-model-%d", i))
	}

	tokenEncoderMutex.RLock()
	cacheSize := len(tokenEncoderMap)
	tokenEncoderMutex.RUnlock()
	if cacheSize != 0 {
		b.Fatalf("unknown model cache size = %d, want 0", cacheSize)
	}
}
