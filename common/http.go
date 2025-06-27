package common

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

func CloseResponseBodyGracefully(httpResponse *http.Response) {
	if httpResponse == nil || httpResponse.Body == nil {
		return
	}
	err := httpResponse.Body.Close()
	if err != nil {
		SysError("failed to close response body: " + err.Error())
	}
}

func IOCopyBytesGracefully(c *gin.Context, src *http.Response, data []byte) {
	if src == nil || src.Body == nil {
		return
	}

	defer CloseResponseBodyGracefully(src)

	if c.Writer == nil {
		return
	}

	src.Body = io.NopCloser(bytes.NewBuffer(data))

	// We shouldn't set the header before we parse the response body, because the parse part may fail.
	// And then we will have to send an error response, but in this case, the header has already been set.
	// So the httpClient will be confused by the response.
	// For example, Postman will report error, and we cannot check the response at all.
	for k, v := range src.Header {
		// avoid setting Content-Length
		if k == "Content-Length" {
			continue
		}
		c.Writer.Header().Set(k, v[0])
	}

	// set Content-Length header manually
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

	c.Writer.WriteHeader(src.StatusCode)
	c.Writer.WriteHeaderNow()

	_, err := io.Copy(c.Writer, src.Body)
	if err != nil {
		LogError(c, fmt.Sprintf("failed to copy response body: %s", err.Error()))
	}
}
