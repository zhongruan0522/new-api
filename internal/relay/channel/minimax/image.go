package minimax

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"
	"github.com/NookMux/NookMux/internal/service"

	"github.com/NookMux/NookMux/pkg/jsonx"
	"github.com/gin-gonic/gin"
)

type MiniMaxImageRequest struct {
	Model           string `json:"model"`
	Prompt          string `json:"prompt"`
	AspectRatio     string `json:"aspect_ratio,omitempty"`
	ResponseFormat  string `json:"response_format,omitempty"`
	N               int    `json:"n,omitempty"`
	PromptOptimizer *bool  `json:"prompt_optimizer,omitempty"`
	AigcWatermark   *bool  `json:"aigc_watermark,omitempty"`
}

type MiniMaxImageResponse struct {
	ID   string `json:"id"`
	Data struct {
		ImageURLs   []string `json:"image_urls"`
		ImageBase64 []string `json:"image_base64"`
	} `json:"data"`
	Metadata map[string]any  `json:"metadata"`
	BaseResp MiniMaxBaseResp `json:"base_resp"`
}

func oaiImage2MiniMaxImageRequest(request shared.ImageRequest) MiniMaxImageRequest {
	responseFormat := normalizeMiniMaxResponseFormat(request.ResponseFormat)
	minimaxRequest := MiniMaxImageRequest{
		Model:          request.Model,
		Prompt:         request.Prompt,
		ResponseFormat: responseFormat,
		N:              1,
		AigcWatermark:  request.Watermark,
	}

	if minimaxRequest.Model == "" {
		minimaxRequest.Model = "image-01"
	}
	if request.N > 0 {
		minimaxRequest.N = int(request.N)
	}
	if aspectRatio := aspectRatioFromImageRequest(request); aspectRatio != "" {
		minimaxRequest.AspectRatio = aspectRatio
	}
	if raw, ok := request.Extra["prompt_optimizer"]; ok {
		var promptOptimizer bool
		if err := jsonx.Unmarshal(raw, &promptOptimizer); err == nil {
			minimaxRequest.PromptOptimizer = &promptOptimizer
		}
	}

	return minimaxRequest
}

func aspectRatioFromImageRequest(request shared.ImageRequest) string {
	if raw, ok := request.Extra["aspect_ratio"]; ok {
		var aspectRatio string
		if err := jsonx.Unmarshal(raw, &aspectRatio); err == nil && aspectRatio != "" {
			return aspectRatio
		}
	}

	switch request.Size {
	case "1024x1024":
		return "1:1"
	case "1792x1024":
		return "16:9"
	case "1024x1792":
		return "9:16"
	case "1536x1024", "1248x832":
		return "3:2"
	case "1024x1536", "832x1248":
		return "2:3"
	case "1152x864":
		return "4:3"
	case "864x1152":
		return "3:4"
	case "1344x576":
		return "21:9"
	}

	width, height, ok := parseImageSize(request.Size)
	if !ok {
		return ""
	}
	ratio := reduceAspectRatio(width, height)
	switch ratio {
	case "1:1", "16:9", "4:3", "3:2", "2:3", "3:4", "9:16", "21:9":
		return ratio
	default:
		return ""
	}
}

func parseImageSize(size string) (int, int, bool) {
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func reduceAspectRatio(width, height int) string {
	divisor := gcd(width, height)
	return fmt.Sprintf("%d:%d", width/divisor, height/divisor)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}

func normalizeMiniMaxResponseFormat(responseFormat string) string {
	switch strings.ToLower(responseFormat) {
	case "", "url":
		return "url"
	case "b64_json", "base64":
		return "base64"
	default:
		return responseFormat
	}
}

func responseMiniMax2OpenAIImage(response *MiniMaxImageResponse, info *relaycommon.RelayInfo) (*shared.ImageResponse, error) {
	imageResponse := &shared.ImageResponse{}
	if info != nil && !info.StartTime.IsZero() {
		imageResponse.Created = info.StartTime.Unix()
	}

	for _, imageURL := range response.Data.ImageURLs {
		imageResponse.Data = append(imageResponse.Data, shared.ImageData{Url: imageURL})
	}
	for _, imageBase64 := range response.Data.ImageBase64 {
		imageResponse.Data = append(imageResponse.Data, shared.ImageData{B64Json: imageBase64})
	}
	if len(response.Metadata) > 0 {
		metadata, err := jsonx.Marshal(response.Metadata)
		if err != nil {
			return nil, err
		}
		imageResponse.Metadata = metadata
	}

	return imageResponse, nil
}

func miniMaxImageHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*shared.Usage, *shared.NookMuxError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := common.ReadMediaResponseBody(resp.Body)
	if err != nil {
		return nil, shared.NewOpenAIError(err, shared.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var minimaxResponse MiniMaxImageResponse
	if err := jsonx.Unmarshal(responseBody, &minimaxResponse); err != nil {
		return nil, shared.NewOpenAIError(err, shared.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if minimaxResponse.BaseResp.StatusCode != 0 {
		return nil, shared.WithOpenAIError(shared.OpenAIError{
			Message: minimaxResponse.BaseResp.StatusMsg,
			Type:    "minimax_image_error",
			Code:    fmt.Sprintf("%d", minimaxResponse.BaseResp.StatusCode),
		}, resp.StatusCode)
	}

	openAIResponse, convertErr := responseMiniMax2OpenAIImage(&minimaxResponse, info)
	if convertErr != nil {
		return nil, shared.NewError(convertErr, shared.ErrorCodeBadResponseBody)
	}
	jsonResponse, err := jsonx.Marshal(openAIResponse)
	if err != nil {
		return nil, shared.NewError(err, shared.ErrorCodeBadResponseBody)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	service.IOCopyBytesGracefully(c, resp, jsonResponse)

	return &shared.Usage{}, nil
}
