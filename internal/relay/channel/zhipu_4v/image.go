package zhipu_4v

import (
	"net/http"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/internal/domain/shared"
	"github.com/NookMux/NookMux/internal/infra/log"
	relaycommon "github.com/NookMux/NookMux/internal/relay/common"

	"github.com/NookMux/NookMux/pkg/jsonx"

	media "github.com/NookMux/NookMux/internal/infra/media"
	"github.com/NookMux/NookMux/internal/relay/helper"
	"github.com/gin-gonic/gin"
)

type zhipuImageResponse struct {
	Created       *int64            `json:"created,omitempty"`
	Data          []zhipuImageData  `json:"data,omitempty"`
	ContentFilter any               `json:"content_filter,omitempty"`
	Usage         *shared.Usage     `json:"usage,omitempty"`
	Error         *zhipuImageError  `json:"error,omitempty"`
	RequestID     string            `json:"request_id,omitempty"`
	ExtendParam   map[string]string `json:"extendParam,omitempty"`
}

type zhipuImageError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type zhipuImageData struct {
	Url      string `json:"url,omitempty"`
	ImageUrl string `json:"image_url,omitempty"`
	B64Json  string `json:"b64_json,omitempty"`
	B64Image string `json:"b64_image,omitempty"`
}

type openAIImagePayload struct {
	Created int64             `json:"created"`
	Data    []openAIImageData `json:"data"`
}

type openAIImageData struct {
	B64Json string `json:"b64_json"`
}

func zhipu4vImageHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*shared.Usage, *shared.NookMuxError) {
	defer helper.CloseResponseBodyGracefully(resp)

	responseBody, err := common.ReadMediaResponseBody(resp.Body)
	if err != nil {
		return nil, shared.NewOpenAIError(err, shared.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var zhipuResp zhipuImageResponse
	if err := jsonx.Unmarshal(responseBody, &zhipuResp); err != nil {
		return nil, shared.NewOpenAIError(err, shared.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	if zhipuResp.Error != nil && zhipuResp.Error.Message != "" {
		return nil, shared.WithOpenAIError(shared.OpenAIError{
			Message: zhipuResp.Error.Message,
			Type:    "zhipu_image_error",
			Code:    zhipuResp.Error.Code,
		}, resp.StatusCode)
	}

	payload := openAIImagePayload{}
	if zhipuResp.Created != nil && *zhipuResp.Created != 0 {
		payload.Created = *zhipuResp.Created
	} else {
		payload.Created = info.StartTime.Unix()
	}
	for _, data := range zhipuResp.Data {
		url := data.Url
		if url == "" {
			url = data.ImageUrl
		}
		if url == "" {
			log.LogWarn(c, "zhipu_image_missing_url")
			continue
		}

		var b64 string
		switch {
		case data.B64Json != "":
			b64 = data.B64Json
		case data.B64Image != "":
			b64 = data.B64Image
		default:
			_, downloaded, err := media.GetImageFromUrl(url)
			if err != nil {
				log.LogError(c, "zhipu_image_get_b64_failed: "+err.Error())
				continue
			}
			b64 = downloaded
		}

		if b64 == "" {
			log.LogWarn(c, "zhipu_image_empty_b64")
			continue
		}

		imageData := openAIImageData{
			B64Json: b64,
		}
		payload.Data = append(payload.Data, imageData)
	}

	jsonResp, err := jsonx.Marshal(payload)
	if err != nil {
		return nil, shared.NewError(err, shared.ErrorCodeBadResponseBody)
	}

	helper.IOCopyBytesGracefully(c, resp, jsonResp)

	return &shared.Usage{}, nil
}
