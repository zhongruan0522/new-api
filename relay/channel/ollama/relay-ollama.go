package ollama

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

func FetchOllamaModels(baseURL, apiKey string) ([]OllamaModel, error) {
	trimmedBase := strings.TrimRight(baseURL, "/")
	url := fmt.Sprintf("%s/v1/models", trimmedBase)

	client := &http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 模型查询改走 OpenAI 兼容接口，便于与新的标准转发链路保持一致。
	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("服务器返回错误 %d: %s", response.StatusCode, string(body))
	}

	var listResponse OllamaOpenAIModelListResponse
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	err = common.Unmarshal(body, &listResponse)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	models := make([]OllamaModel, 0, len(listResponse.Data))
	for _, item := range listResponse.Data {
		model := OllamaModel{
			Name:    item.ID,
			Created: item.Created,
			OwnedBy: item.OwnedBy,
		}
		if item.Created > 0 {
			model.ModifiedAt = time.Unix(item.Created, 0).UTC().Format(time.RFC3339)
		}
		models = append(models, model)
	}

	return models, nil
}

// 拉取 Ollama 模型 (非流式)
func PullOllamaModel(baseURL, apiKey, modelName string) error {
	url := fmt.Sprintf("%s/api/pull", baseURL)

	pullRequest := OllamaPullRequest{
		Name:   modelName,
		Stream: false, // 非流式，简化处理
	}

	requestBody, err := common.Marshal(pullRequest)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{
		Timeout: 30 * 60 * 1000 * time.Millisecond, // 30分钟超时，支持大模型
	}
	request, err := http.NewRequest("POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("拉取模型失败 %d: %s", response.StatusCode, string(body))
	}

	return nil
}

// 流式拉取 Ollama 模型 (支持进度回调)
func PullOllamaModelStream(baseURL, apiKey, modelName string, progressCallback func(OllamaPullResponse)) error {
	url := fmt.Sprintf("%s/api/pull", baseURL)

	pullRequest := OllamaPullRequest{
		Name:   modelName,
		Stream: true, // 启用流式
	}

	requestBody, err := common.Marshal(pullRequest)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{
		Timeout: 60 * 60 * 1000 * time.Millisecond, // 1小时超时，支持超大模型
	}
	request, err := http.NewRequest("POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("拉取模型失败 %d: %s", response.StatusCode, string(body))
	}

	// 读取流式响应
	scanner := bufio.NewScanner(response.Body)
	successful := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var pullResponse OllamaPullResponse
		if err := common.Unmarshal([]byte(line), &pullResponse); err != nil {
			continue // 忽略解析失败的行
		}

		if progressCallback != nil {
			progressCallback(pullResponse)
		}

		// 检查是否出现错误或完成
		if strings.EqualFold(pullResponse.Status, "error") {
			return fmt.Errorf("拉取模型失败: %s", strings.TrimSpace(line))
		}
		if strings.EqualFold(pullResponse.Status, "success") {
			successful = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取流式响应失败: %v", err)
	}

	if !successful {
		return fmt.Errorf("拉取模型未完成: 未收到成功状态")
	}

	return nil
}

// 删除 Ollama 模型
func DeleteOllamaModel(baseURL, apiKey, modelName string) error {
	url := fmt.Sprintf("%s/api/delete", baseURL)

	deleteRequest := OllamaDeleteRequest{
		Name: modelName,
	}

	requestBody, err := common.Marshal(deleteRequest)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{}
	request, err := http.NewRequest("DELETE", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("删除模型失败 %d: %s", response.StatusCode, string(body))
	}

	return nil
}

func FetchOllamaVersion(baseURL, apiKey string) (string, error) {
	trimmedBase := strings.TrimRight(baseURL, "/")
	if trimmedBase == "" {
		return "", fmt.Errorf("baseURL 为空")
	}

	url := fmt.Sprintf("%s/api/version", trimmedBase)

	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("查询版本失败 %d: %s", response.StatusCode, string(body))
	}

	var versionResp struct {
		Version string `json:"version"`
	}

	if err := common.Unmarshal(body, &versionResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if versionResp.Version == "" {
		return "", fmt.Errorf("未返回版本信息")
	}

	return versionResp.Version, nil
}
