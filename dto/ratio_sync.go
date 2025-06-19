package dto

// UpstreamDTO 提交到后端同步倍率的上游渠道信息
// Endpoint 可以为空，后端会默认使用 /api/ratio_config
// BaseURL 必须以 http/https 开头，不要以 / 结尾
// 例如： https://api.example.com
// Endpoint: /api/ratio_config
// 提交示例：
// {
//   "name": "openai",
//   "base_url": "https://api.openai.com",
//   "endpoint": "/ratio_config"
// }

type UpstreamDTO struct {
    Name     string `json:"name" binding:"required"`
    BaseURL  string `json:"base_url" binding:"required"`
    Endpoint string `json:"endpoint"`
}

type UpstreamRequest struct {
    ChannelIDs []int64 `json:"channel_ids"`
    Timeout    int     `json:"timeout"`
}

// TestResult 上游测试连通性结果
type TestResult struct {
    Name   string `json:"name"`
    Status string `json:"status"`
    Error  string `json:"error,omitempty"`
}

// DifferenceItem 差异项
// Current 为本地值，可能为 nil
// Upstreams 为各渠道的上游值，具体数值 / "same" / nil

type DifferenceItem struct {
    Current   interface{}            `json:"current"`
    Upstreams map[string]interface{} `json:"upstreams"`
}

// SyncableChannel 可同步的渠道信息（base_url 不为空）

type SyncableChannel struct {
    ID      int    `json:"id"`
    Name    string `json:"name"`
    BaseURL string `json:"base_url"`
    Status  int    `json:"status"`
} 