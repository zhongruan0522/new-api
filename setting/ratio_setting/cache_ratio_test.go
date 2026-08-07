package ratio_setting

import "testing"

func TestDefaultGemini3CacheRatios(t *testing.T) {
	previous := GetCacheRatioCopy()
	cacheRatioMap.Clear()
	t.Cleanup(func() {
		cacheRatioMap.Clear()
		cacheRatioMap.AddAll(previous)
	})

	InitRatioSettings()

	for _, model := range []string{
		"gemini-3-flash-preview",
		"gemini-3.1-pro-preview",
	} {
		t.Run(model, func(t *testing.T) {
			ratio, ok := GetCacheRatio(model)
			if !ok {
				t.Fatalf("GetCacheRatio(%q) did not find a default ratio", model)
			}
			if ratio != 0.1 {
				t.Fatalf("GetCacheRatio(%q) = %v, want 0.1", model, ratio)
			}
		})
	}
}
