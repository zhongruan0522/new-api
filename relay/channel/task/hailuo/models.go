package hailuo

type TextToVideoRequest struct {
	Model            string `json:"model"`
	Prompt           string `json:"prompt"`
	PromptOptimizer  *bool  `json:"prompt_optimizer,omitempty"`
	FastPretreatment *bool  `json:"fast_pretreatment,omitempty"`
	Duration         *int   `json:"duration,omitempty"`
	Resolution       string `json:"resolution,omitempty"`
	CallbackURL      string `json:"callback_url,omitempty"`
	AigcWatermark    *bool  `json:"aigc_watermark,omitempty"`
}

type TextToVideoResponse struct {
	TaskID   string   `json:"task_id"`
	BaseResp BaseResp `json:"base_resp"`
}

type BaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type QueryTaskRequest struct {
	TaskID string `json:"task_id"`
}

type QueryTaskResponse struct {
	TaskID   string   `json:"task_id"`
	Status   string   `json:"status"`
	FileID   string   `json:"file_id,omitempty"`
	VideoURL string   `json:"video_url,omitempty"`
	BaseResp BaseResp `json:"base_resp"`
}

type ErrorInfo struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type TaskStatusInfo struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	FileID    string `json:"file_id,omitempty"`
	VideoURL  string `json:"video_url,omitempty"`
	ErrorCode int    `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

type ModelConfig struct {
	Name                 string
	DefaultResolution    string
	SupportedDurations   []int
	SupportedResolutions []string
	HasPromptOptimizer   bool
	HasFastPretreatment  bool
}

func GetModelConfig(model string) ModelConfig {
	configs := map[string]ModelConfig{
		"MiniMax-Hailuo-2.3": {
			Name:                 "MiniMax-Hailuo-2.3",
			DefaultResolution:    Resolution768P,
			SupportedDurations:   []int{6, 10},
			SupportedResolutions: []string{Resolution768P, Resolution1080P},
			HasPromptOptimizer:   true,
			HasFastPretreatment:  true,
		},
		"MiniMax-Hailuo-02": {
			Name:                 "MiniMax-Hailuo-02",
			DefaultResolution:    Resolution768P,
			SupportedDurations:   []int{6, 10},
			SupportedResolutions: []string{Resolution768P, Resolution1080P},
			HasPromptOptimizer:   true,
			HasFastPretreatment:  true,
		},
		"T2V-01-Director": {
			Name:                 "T2V-01-Director",
			DefaultResolution:    Resolution768P,
			SupportedDurations:   []int{6},
			SupportedResolutions: []string{Resolution768P, Resolution1080P},
			HasPromptOptimizer:   true,
			HasFastPretreatment:  false,
		},
		"T2V-01": {
			Name:                 "T2V-01",
			DefaultResolution:    Resolution720P,
			SupportedDurations:   []int{6},
			SupportedResolutions: []string{Resolution720P},
			HasPromptOptimizer:   true,
			HasFastPretreatment:  false,
		},
	}

	if config, exists := configs[model]; exists {
		return config
	}

	return ModelConfig{
		Name:                 model,
		DefaultResolution:    Resolution720P,
		SupportedDurations:   []int{6},
		SupportedResolutions: []string{Resolution720P},
		HasPromptOptimizer:   true,
		HasFastPretreatment:  false,
	}
}
