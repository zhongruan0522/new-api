package jimeng

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"one-api/common"
	"one-api/dto"
	relaycommon "one-api/relay/common"
	"one-api/service"
)

type ImageResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		BinaryDataBase64 []string `json:"binary_data_base64"`
		ImageUrls        []string `json:"image_urls"`
		RephraseResult   string   `json:"rephraser_result"`
		RequestID        string   `json:"request_id"`
		// Other fields are omitted for brevity
	} `json:"data"`
	RequestID   string `json:"request_id"`
	Status      int    `json:"status"`
	TimeElapsed string `json:"time_elapsed"`
}

func responseJimeng2OpenAIImage(_ *gin.Context, response *ImageResponse, info *relaycommon.RelayInfo) *dto.ImageResponse {
	imageResponse := dto.ImageResponse{
		Created: info.StartTime.Unix(),
	}

	for _, base64Data := range response.Data.BinaryDataBase64 {
		imageResponse.Data = append(imageResponse.Data, dto.ImageData{
			B64Json: base64Data,
		})
	}
	for _, imageUrl := range response.Data.ImageUrls {
		imageResponse.Data = append(imageResponse.Data, dto.ImageData{
			Url: imageUrl,
		})
	}

	return &imageResponse
}

// jimengImageHandler handles the Jimeng image generation response
func jimengImageHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.OpenAIErrorWithStatusCode, *dto.Usage) {
	var jimengResponse ImageResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	common.CloseResponseBodyGracefully(resp)

	err = json.Unmarshal(responseBody, &jimengResponse)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}

	// Check if the response indicates an error
	if jimengResponse.Code != 10000 {
		return &dto.OpenAIErrorWithStatusCode{
			Error: dto.OpenAIError{
				Message: jimengResponse.Message,
				Type:    "jimeng_error",
				Param:   "",
				Code:    fmt.Sprintf("%d", jimengResponse.Code),
			},
			StatusCode: resp.StatusCode,
		}, nil
	}

	// Convert Jimeng response to OpenAI format
	fullTextResponse := responseJimeng2OpenAIImage(c, &jimengResponse, info)
	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	if err != nil {
		return service.OpenAIErrorWrapper(err, "write_response_body_failed", http.StatusInternalServerError), nil
	}

	return nil, &dto.Usage{}
}
