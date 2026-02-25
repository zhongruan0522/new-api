package controller

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type storedMediaListRow struct {
	Id        string `json:"id"`
	MediaType string `json:"media_type"`
	CreatedAt int64  `json:"created_at"`
	MimeType  string `json:"mime_type"`
	SizeBytes int    `json:"size_bytes"`
	Url       string `json:"url"`
}

type storedMediaDetailResponse struct {
	Id           string `json:"id"`
	MediaType    string `json:"media_type"`
	CreatedAt    int64  `json:"created_at"`
	MimeType     string `json:"mime_type"`
	SizeBytes    int    `json:"size_bytes"`
	Url          string `json:"url"`
	BasePreview  string `json:"base_preview"`
	BaseTruncate bool   `json:"base_truncated"`
}

type storedMediaBatchItem struct {
	Id        string `json:"id"`
	MediaType string `json:"media_type"`
}

type storedMediaBatchRequest struct {
	Items []storedMediaBatchItem `json:"items"`
}

func GetAllStoredMedia(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	items, total, err := model.GetAllStoredMedia(c.Request.Context(), startTimestamp, endTimestamp, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	rows := make([]storedMediaListRow, 0, len(items))
	for i := range items {
		rows = append(rows, storedMediaListRow{
			Id:        items[i].Id,
			MediaType: items[i].MediaType,
			CreatedAt: items[i].CreatedAt,
			MimeType:  items[i].MimeType,
			SizeBytes: items[i].SizeBytes,
			Url:       buildStoredMediaURL(c, items[i].MediaType, items[i].Id),
		})
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(rows)
	common.ApiSuccess(c, pageInfo)
}

func GetSelfStoredMedia(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	items, total, err := model.GetUserStoredMedia(c.Request.Context(), userId, startTimestamp, endTimestamp, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	rows := make([]storedMediaListRow, 0, len(items))
	for i := range items {
		rows = append(rows, storedMediaListRow{
			Id:        items[i].Id,
			MediaType: items[i].MediaType,
			CreatedAt: items[i].CreatedAt,
			MimeType:  items[i].MimeType,
			SizeBytes: items[i].SizeBytes,
			Url:       buildStoredMediaURL(c, items[i].MediaType, items[i].Id),
		})
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(rows)
	common.ApiSuccess(c, pageInfo)
}

func GetStoredMediaDetail(c *gin.Context) {
	mediaType := strings.TrimSpace(strings.ToLower(c.Param("media_type")))
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		common.ApiErrorMsg(c, "id is required")
		return
	}
	if mediaType != "image" && mediaType != "video" {
		common.ApiErrorMsg(c, "media_type must be image or video")
		return
	}

	userId := c.GetInt("id")
	role := c.GetInt("role")
	isAdminUser := role >= common.RoleAdminUser

	const previewBytes = 32 * 1024

	if mediaType == "image" {
		meta, err := model.GetStoredImageMetaByID(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				common.ApiErrorMsg(c, "not found")
				return
			}
			common.ApiError(c, err)
			return
		}
		if !isAdminUser && meta.UserId != userId {
			common.ApiErrorMsg(c, "forbidden")
			return
		}

		img, err := model.GetStoredImageByID(c.Request.Context(), id)
		if err != nil {
			common.ApiError(c, err)
			return
		}

		basePreview, truncated := buildDataURLBase64Preview(img.MimeType, []byte(img.Data), previewBytes)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": storedMediaDetailResponse{
				Id:           img.Id,
				MediaType:    "image",
				CreatedAt:    img.CreatedAt,
				MimeType:     img.MimeType,
				SizeBytes:    img.SizeBytes,
				Url:          buildStoredMediaURL(c, "image", img.Id),
				BasePreview:  basePreview,
				BaseTruncate: truncated,
			},
		})
		return
	}

	meta, err := model.GetStoredVideoMetaByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	if !isAdminUser && meta.UserId != userId {
		common.ApiErrorMsg(c, "forbidden")
		return
	}

	v, err := model.GetStoredVideoByID(c.Request.Context(), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	basePreview, truncated := buildDataURLBase64Preview(v.MimeType, []byte(v.Data), previewBytes)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": storedMediaDetailResponse{
			Id:           v.Id,
			MediaType:    "video",
			CreatedAt:    v.CreatedAt,
			MimeType:     v.MimeType,
			SizeBytes:    v.SizeBytes,
			Url:          buildStoredMediaURL(c, "video", v.Id),
			BasePreview:  basePreview,
			BaseTruncate: truncated,
		},
	})
}

func DeleteStoredMedia(c *gin.Context) {
	mediaType := strings.TrimSpace(strings.ToLower(c.Param("media_type")))
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		common.ApiErrorMsg(c, "id is required")
		return
	}
	if mediaType != "image" && mediaType != "video" {
		common.ApiErrorMsg(c, "media_type must be image or video")
		return
	}

	userId := c.GetInt("id")
	role := c.GetInt("role")
	isAdminUser := role >= common.RoleAdminUser

	var deleted int64
	var err error

	if mediaType == "image" {
		meta, metaErr := model.GetStoredImageMetaByID(c.Request.Context(), id)
		if metaErr != nil {
			if errors.Is(metaErr, gorm.ErrRecordNotFound) {
				common.ApiErrorMsg(c, "not found")
				return
			}
			common.ApiError(c, metaErr)
			return
		}
		if !isAdminUser && meta.UserId != userId {
			common.ApiErrorMsg(c, "forbidden")
			return
		}
		if isAdminUser {
			deleted, err = model.DeleteStoredImagesByIDs(c.Request.Context(), []string{id}, 0)
		} else {
			deleted, err = model.DeleteStoredImagesByIDs(c.Request.Context(), []string{id}, userId)
		}
	} else {
		meta, metaErr := model.GetStoredVideoMetaByID(c.Request.Context(), id)
		if metaErr != nil {
			if errors.Is(metaErr, gorm.ErrRecordNotFound) {
				common.ApiErrorMsg(c, "not found")
				return
			}
			common.ApiError(c, metaErr)
			return
		}
		if !isAdminUser && meta.UserId != userId {
			common.ApiErrorMsg(c, "forbidden")
			return
		}
		if isAdminUser {
			deleted, err = model.DeleteStoredVideosByIDs(c.Request.Context(), []string{id}, 0)
		} else {
			deleted, err = model.DeleteStoredVideosByIDs(c.Request.Context(), []string{id}, userId)
		}
	}

	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    deleted,
	})
}

func DeleteStoredMediaBatch(c *gin.Context) {
	req := storedMediaBatchRequest{}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
		common.ApiErrorMsg(c, "invalid params")
		return
	}

	userId := c.GetInt("id")
	role := c.GetInt("role")
	isAdminUser := role >= common.RoleAdminUser

	imageIDs := make([]string, 0, len(req.Items))
	videoIDs := make([]string, 0, len(req.Items))

	for i := range req.Items {
		id := strings.TrimSpace(req.Items[i].Id)
		typ := strings.TrimSpace(strings.ToLower(req.Items[i].MediaType))
		if id == "" {
			continue
		}
		switch typ {
		case "image":
			imageIDs = append(imageIDs, id)
		case "video":
			videoIDs = append(videoIDs, id)
		default:
			// ignore unknown types
		}
	}

	if len(imageIDs) == 0 && len(videoIDs) == 0 {
		common.ApiErrorMsg(c, "no valid ids")
		return
	}

	var totalDeleted int64 = 0

	imgUser := userId
	videoUser := userId
	if isAdminUser {
		imgUser = 0
		videoUser = 0
	}

	if len(imageIDs) > 0 {
		n, err := model.DeleteStoredImagesByIDs(c.Request.Context(), imageIDs, imgUser)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		totalDeleted += n
	}
	if len(videoIDs) > 0 {
		n, err := model.DeleteStoredVideosByIDs(c.Request.Context(), videoIDs, videoUser)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		totalDeleted += n
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    totalDeleted,
	})
}

func buildDataURLBase64Preview(mimeType string, data []byte, previewBytes int) (string, bool) {
	mimeType = strings.TrimSpace(mimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if previewBytes <= 0 {
		previewBytes = 32 * 1024
	}

	truncated := false
	if len(data) > previewBytes {
		data = data[:previewBytes]
		truncated = true
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), truncated
}

func buildStoredMediaURL(c *gin.Context, mediaType string, id string) string {
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}

	var scope string
	switch mediaType {
	case "image":
		scope = "stored_image"
	case "video":
		scope = "stored_video"
	default:
		return ""
	}

	sig := common.GenerateHMAC(fmt.Sprintf("%s:%s", scope, id))
	path := fmt.Sprintf("/mcp/%s/%s?sig=%s", mediaType, url.PathEscape(id), sig)

	base := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	if base == "" {
		base = guessBaseURLFromRequest(c)
	}
	if base == "" {
		return path
	}
	return base + path
}

func guessBaseURLFromRequest(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	proto := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	host := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = c.Request.Host
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	return proto + "://" + host
}
