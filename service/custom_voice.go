package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhongruan0522/new-api/common"
	"github.com/zhongruan0522/new-api/model"
	"github.com/zhongruan0522/new-api/setting/model_setting"
	"github.com/zhongruan0522/new-api/setting/ratio_setting"
)

// 定制音色流程的常量与校验规则。
const (
	customVoiceMaxFileSize    = 20 << 20 // 20MB，与旧版一致
	customVoicePreviewTextMax = 2000
	customVoicePreviewTimeout = 90 * time.Second
	// customVoicePreviewTTL 试听记录保留时长：超过该时长未确认的“试听中”记录将被自动清理。
	// 业务规则：7 天内未确认的试听视为放弃，直接删除记录（不写审计日志）。
	customVoicePreviewTTL = 7 * 24 * time.Hour
)

var (
	customVoiceIDPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*[^-_]$`)

	// 允许上传的音频扩展名。旧版仅做前端 accept 限制，这里服务端强制校验。
	customVoiceAllowedExts = map[string]struct{}{
		".mp3": {},
		".m4a": {},
		".wav": {},
	}
)

// CustomVoicePreviewRequest 试听（上传并克隆）请求参数。
type CustomVoicePreviewRequest struct {
	Model                 string // 客户端选择的 TTS 模型，用于按所选模型计费
	VoiceId               string // 用户填写的音色 ID
	PreviewText           string // 可选试听文本
	NeedNoiseReduction    bool
	NeedVolumeNormalization bool
}

// CustomVoicePreviewResult 试听返回结果。
type CustomVoicePreviewResult struct {
	VoiceId    string `json:"voice_id"`
	DemoAudio  string `json:"demo_audio"`   // 试听音频 URL（上游返回）
	FileId     string `json:"file_id"`      // 上游文件 ID（便于调试，不回传敏感信息）
	RecordId   int64  `json:"record_id"`    // 写入的试听中记录 ID
}

// CustomVoiceConfirmResult 确认定制返回结果。
type CustomVoiceConfirmResult struct {
	VoiceId string `json:"voice_id"`
	Status  string `json:"status"`
}

// validateCustomVoiceID 校验音色 ID：长度 8-256，字母开头，字母数字/下划线/连字符，不能以 _ - 结尾。
func validateCustomVoiceID(voiceId string) error {
	voiceId = strings.TrimSpace(voiceId)
	if len(voiceId) < 8 || len(voiceId) > 256 {
		return errors.New("音色ID不合规")
	}
	if !customVoiceIDPattern.MatchString(voiceId) {
		return errors.New("音色ID不合规")
	}
	return nil
}

// validateCustomVoiceFile 校验上传文件大小与扩展名。
func validateCustomVoiceFile(header *multipart.FileHeader) error {
	if header == nil {
		return errors.New("请上传音频文件")
	}
	if header.Size <= 0 {
		return errors.New("音频文件为空")
	}
	if header.Size > customVoiceMaxFileSize {
		return errors.New("音频文件过大，请压缩到 20MB 以内")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if _, ok := customVoiceAllowedExts[ext]; !ok {
		return errors.New("仅支持 mp3、m4a、wav 格式")
	}
	return nil
}

// minimaxUpstream 是从渠道解析出的上游调用所需信息。
type minimaxUpstream struct {
	baseURL  string
	apiKey   string
	groupId  string
	channel  *model.Channel
}

// resolveMiniMaxUpstream 解析定制音色分组的上游 MiniMax 渠道信息。
// group 来自系统设置 CustomVoiceGroup；GroupId 从渠道 Other 字段读取（管理员填写）。
func resolveMiniMaxUpstream(group string) (*minimaxUpstream, error) {
	group = strings.TrimSpace(group)
	ch, err := model.GetEnabledMiniMaxChannelForGroup(group)
	if err != nil || ch == nil {
		return nil, errors.New("未找到可用的渠道，请联系管理员")
	}
	baseURL := ""
	if ch.BaseURL != nil {
		baseURL = strings.TrimRight(*ch.BaseURL, "/")
	}
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com/v1"
	}
	// MiniMax GroupId 存放在渠道 Other 字段（管理员填写）。
	groupId := strings.TrimSpace(ch.Other)
	keys := ch.GetKeys()
	if len(keys) == 0 || keys[0] == "" {
		return nil, errors.New("渠道凭证缺失，请联系管理员")
	}
	return &minimaxUpstream{
		baseURL: baseURL,
		apiKey:  keys[0],
		groupId: groupId,
		channel: ch,
	}, nil
}

// doUpstreamRequest 执行上游 HTTP 请求并返回状态码与响应体。
func doUpstreamRequest(url string, contentType string, body io.Reader, apiKey string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client := &http.Client{Timeout: customVoicePreviewTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, errors.New("上游服务暂不可用，请稍后重试")
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if readErr != nil {
		return resp.StatusCode, nil, errors.New("读取上游响应失败")
	}
	return resp.StatusCode, data, nil
}

// upstreamBaseResp 兼容 MiniMax 返回的 base_resp 结构，用于判定成功/失败。
type upstreamBaseResp struct {
	BaseResp struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
}

// normalizeUpstreamError 把上游错误转成面向用户的普通业务提示，不暴露渠道名。
// status 为 HTTP 状态码，rawBody 为上游响应体。
func normalizeUpstreamError(status int, rawBody []byte) error {
	msg := "音色处理失败，请稍后重试"
	if status >= 500 {
		return errors.New("上游服务繁忙，请稍后重试")
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return errors.New("服务凭证无效，请联系管理员")
	}
	if status == http.StatusTooManyRequests {
		return errors.New("请求过于频繁，请稍后重试")
	}
	// 尝试解析上游 base_resp，进一步精简提示，但绝不回传完整原始信息。
	var br upstreamBaseResp
	if err := common.Unmarshal(rawBody, &br); err == nil {
		if br.BaseResp.StatusCode != 0 {
			inner := strings.TrimSpace(br.BaseResp.StatusMsg)
			if inner != "" {
				// 仅保留简短摘要，避免泄露内部细节。
				if len([]rune(inner)) > 40 {
					inner = string([]rune(inner)[:40]) + "..."
				}
				msg = "音色处理失败：" + inner
			}
		}
	}
	return errors.New(msg)
}

// chargeModelOnce 按模型 ID 扣费一次（基于 ModelPrice，乘以分组倍率）。
// 用于：试听阶段按所选 TTS 模型计费、确认定制阶段按配置的扣费模型 ID 计费。
//
// 计费原则（防越权/防漏扣）：
//   - 必须能解析出有效价格（ModelPrice 优先，否则按 ModelRatio 每千 token 折算 1 单位），
//     解析失败则失败即中断，绝不“无扣费成功”。
//   - 仅扣减用户钱包额度（custom voice 流程无 token 上下文），并记录消费日志。
//   - userId/tokenName/channelId 仅用于日志展示。
//
// 返回扣减的 quota。
func chargeModelOnce(c *gin.Context, userId int, channelId int, modelName, group string) (int, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return 0, errors.New("计费模型未配置，请联系管理员")
	}
	price, ok := ratio_setting.GetModelPrice(modelName, false)
	usePrice := ok && price > 0
	quota := 0
	if usePrice {
		quota = int(price * common.QuotaPerUnit)
	} else {
		ratio, ratioOk, _ := ratio_setting.GetModelRatio(modelName)
		if !ratioOk {
			// fail-closed：找不到计费配置时直接拒绝，避免无扣费使用。
			return 0, errors.New("所选计费模型暂不可用，请联系管理员")
		}
		// 按 1 千 token 计算一次调用的基准 quota（ratio 为每千 token 倍率）。
		quota = int(ratio * common.QuotaPerUnit)
	}
	if quota <= 0 {
		// 价格为 0 视为未正确配置，避免免费滥用。
		return 0, errors.New("所选计费模型价格无效，请联系管理员")
	}

	// 预检用户余额，避免无效的上游调用与负数额度。
	remain, err := model.GetUserQuota(userId, true)
	if err != nil {
		return 0, errors.New("额度查询失败，请稍后重试")
	}
	if remain < quota {
		return 0, errors.New("额度不足，请充值后再试")
	}

	if err := model.DecreaseUserQuota(userId, quota); err != nil {
		return 0, errors.New("扣费失败，请稍后重试")
	}
	model.UpdateUserUsedQuotaAndRequestCount(userId, quota)

	// 记录消费日志（按 token 0 的方式记录一次调用）。
	tokenName := c.GetString("token_name")
	model.RecordConsumeLog(c, userId, model.RecordConsumeLogParams{
		ChannelId:   channelId,
		PromptTokens: 0,
		CompletionTokens: 0,
		ModelName:   modelName,
		TokenName:   tokenName,
		Quota:       quota,
		Content:     "音色定制相关消费",
		UseTimeMs:   0,
		IsStream:    false,
		Group:       group,
	})
	return quota, nil
}

// CustomVoicePreview 执行“上传音频 + 生成试听”流程：
//  1. 校验文件、音色 ID；
//  2. 查重音色 ID（已存在则返回泛化的“不合规”提示）；
//  3. 按所选 TTS 模型计费（试听阶段）；
//  4. 上传文件到上游、调用 voice_clone 获取 demo_audio；
//  5. 写入“试听中”音色记录。
//
// 该函数不写审计日志（用户创建音色不审计）。
func CustomVoicePreview(c *gin.Context, userId int, req CustomVoicePreviewRequest, fileHeader *multipart.FileHeader) (*CustomVoicePreviewResult, error) {
	if err := validateCustomVoiceID(req.VoiceId); err != nil {
		return nil, err
	}
	if err := validateCustomVoiceFile(fileHeader); err != nil {
		return nil, err
	}
	if req.PreviewText != "" && len([]rune(req.PreviewText)) > customVoicePreviewTextMax {
		return nil, fmt.Errorf("试听文本过长，最多 %d 字", customVoicePreviewTextMax)
	}

	// 区域开关与渠道解析。
	if !isCustomVoiceConfigReady() {
		return nil, errors.New("定制音色功能未开启，请联系管理员")
	}
	group, _ := getCustomVoiceGroupAndBilling()
	upstream, err := resolveMiniMaxUpstream(group)
	if err != nil {
		return nil, err
	}

	// 先清理超过 7 天未确认的试听记录，确保过期音色 ID 可被重新使用（系统自动清理，不写审计）。
	if err := cleanupExpiredCustomVoicePreviews(); err != nil {
		return nil, err
	}

	// 查重：已存在则提示不合规（不暴露“重复”）。
	exists, err := model.IsMiniMaxVoiceIdExists(req.VoiceId)
	if err != nil {
		return nil, errors.New("音色校验失败，请稍后重试")
	}
	if exists {
		return nil, errors.New("音色ID不合规")
	}

	// 试听阶段按所选 TTS 模型计费。计费失败即中断，不调用上游。
	if _, err := chargeModelOnce(c, userId, upstream.channel.Id, req.Model, group); err != nil {
		return nil, err
	}

	// 上传文件到上游。
	fileId, err := uploadFileUpstream(c, upstream, fileHeader)
	if err != nil {
		return nil, err
	}
	// 调用 voice_clone。
	demoAudio, err := cloneVoiceUpstream(upstream, fileId, req)
	if err != nil {
		return nil, err
	}

	// 写入“试听中”记录（用户创建，不审计）。
	voice := &model.MiniMaxVoice{
		Type:         model.MiniMaxVoiceTypePreview,
		OperatorId:   userId,
		OperatorKind: "user",
		VoiceId:      req.VoiceId,
		Allowed:      false,
	}
	if err := model.InsertMiniMaxVoice(voice); err != nil {
		// 唯一索引冲突也归一为“不合规”，避免暴露重复。
		if isDuplicateKeyErr(err) {
			return nil, errors.New("音色ID不合规")
		}
		return nil, errors.New("音色记录保存失败，请稍后重试")
	}

	return &CustomVoicePreviewResult{
		VoiceId:   req.VoiceId,
		DemoAudio: demoAudio,
		FileId:    fileId,
		RecordId:  voice.Id,
	}, nil
}

// uploadFileUpstream 把用户上传的音频转发到上游文件接口，返回上游 file_id。
func uploadFileUpstream(c *gin.Context, up *minimaxUpstream, header *multipart.FileHeader) (string, error) {
	src, err := header.Open()
	if err != nil {
		return "", errors.New("音频文件读取失败")
	}
	defer src.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	// purpose=voice_clone
	if werr := writer.WriteField("purpose", "voice_clone"); werr != nil {
		return "", errors.New("音频上传准备失败")
	}
	part, err := writer.CreateFormFile("file", header.Filename)
	if err != nil {
		return "", errors.New("音频上传准备失败")
	}
	if _, err := io.Copy(part, src); err != nil {
		return "", errors.New("音频上传准备失败")
	}
	if err := writer.Close(); err != nil {
		return "", errors.New("音频上传准备失败")
	}

	url := up.baseURL + "/files/upload"
	if up.groupId != "" {
		url += "?GroupId=" + up.groupId
	}
	status, body, err := doUpstreamRequest(url, writer.FormDataContentType(), &buf, up.apiKey)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", normalizeUpstreamError(status, body)
	}
	var resp struct {
		File struct {
			FileId string `json:"file_id"`
		} `json:"file"`
		upstreamBaseResp
	}
	if err := common.Unmarshal(body, &resp); err != nil {
		return "", errors.New("上游响应解析失败")
	}
	if resp.BaseResp.StatusCode != 0 || resp.File.FileId == "" {
		return "", normalizeUpstreamError(status, body)
	}
	return resp.File.FileId, nil
}

// cloneVoiceUpstream 调用上游 voice_clone 接口生成试听音频，返回 demo_audio URL。
func cloneVoiceUpstream(up *minimaxUpstream, fileId string, req CustomVoicePreviewRequest) (string, error) {
	payload := map[string]interface{}{
		"file_id":                    fileId,
		"voice_id":                   req.VoiceId,
		"need_noise_reduction":       req.NeedNoiseReduction,
		"need_volume_normalization":  req.NeedVolumeNormalization,
	}
	previewText := strings.TrimSpace(req.PreviewText)
	if previewText != "" {
		payload["text"] = previewText
		payload["model"] = req.Model
	}
	bodyBytes, err := common.Marshal(payload)
	if err != nil {
		return "", errors.New("请求构建失败")
	}

	url := up.baseURL + "/voice_clone"
	if up.groupId != "" {
		url += "?GroupId=" + up.groupId
	}
	status, respBody, err := doUpstreamRequest(url, "application/json", bytes.NewReader(bodyBytes), up.apiKey)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", normalizeUpstreamError(status, respBody)
	}
	var resp struct {
		DemoAudio string `json:"demo_audio"`
		upstreamBaseResp
	}
	if err := common.Unmarshal(respBody, &resp); err != nil {
		return "", errors.New("上游响应解析失败")
	}
	if resp.BaseResp.StatusCode != 0 {
		return "", normalizeUpstreamError(status, respBody)
	}
	return resp.DemoAudio, nil
}

// CustomVoiceConfirm 确认定制：按配置的扣费模型 ID 扣费，成功后把记录从试听中转为已创建。
//
// 安全要点：
//   - 必须命中本用户的“试听中”记录，防止越权确认他人音色。
//   - 一个“试听中”音色 ID 只能确认成功一次：通过条件更新（type=preview -> created）实现幂等。
//   - 扣费失败即中断，记录不会变为已创建。
func CustomVoiceConfirm(c *gin.Context, userId int, voiceId string) (*CustomVoiceConfirmResult, error) {
	voiceId = strings.TrimSpace(voiceId)
	if voiceId == "" {
		return nil, errors.New("音色ID不能为空")
	}
	if !isCustomVoiceConfigReady() {
		return nil, errors.New("定制音色功能未开启，请联系管理员")
	}
	group, billingModel := getCustomVoiceGroupAndBilling()

	// 先清理超过 7 天未确认的试听记录：过期试听不能再确认（系统自动清理，不写审计）。
	if err := cleanupExpiredCustomVoicePreviews(); err != nil {
		return nil, err
	}

	// 必须命中本用户的试听中记录，防止越权。
	voice, err := model.GetMiniMaxVoiceByVoiceId(voiceId)
	if err != nil || voice == nil {
		return nil, errors.New("音色ID不合规")
	}
	if voice.Type != model.MiniMaxVoiceTypePreview {
		return nil, errors.New("该音色无需确认或已处理")
	}
	if voice.OperatorId != userId {
		return nil, errors.New("无权操作该音色")
	}

	// 确认定制阶段按配置的扣费模型 ID 扣费。失败即中断。
	quota, err := chargeModelOnce(c, userId, 0, billingModel, group)
	if err != nil {
		return nil, err
	}

	// 扣费成功后流转状态；记录扣费额度便于审计展示。
	if err := model.UpdateMiniMaxVoiceType(voice.Id, model.MiniMaxVoiceTypeCreated); err != nil {
		// 状态更新失败：尽力退还额度，避免无音色却扣费。
		_ = model.IncreaseUserQuota(userId, quota, false)
		return nil, errors.New("音色激活失败，已退还费用")
	}
	// 回写扣费额度（用户流程不写审计，仅落库）。
	voice.QuotaCost = quota
	_ = model.UpdateMiniMaxVoice(voice)

	return &CustomVoiceConfirmResult{
		VoiceId: voiceId,
		Status:  model.MiniMaxVoiceTypeCreated,
	}, nil
}

// isCustomVoiceConfigReady 检查定制音色区域是否开启。
// 分组允许为空（按默认分组路由），但开关必须打开。
func isCustomVoiceConfigReady() bool {
	return model_setting.IsCustomVoiceEnabled()
}

// cleanupExpiredCustomVoicePreviews 清理超过试听保留时长的“试听中”记录。
// 业务规则：试听阶段超过 customVoicePreviewTTL 仍未确认的记录视为放弃，直接删除。
// 该清理是系统自动行为，不走 controller 删除路径，因此不写审计日志。
// 清理失败时显式返回错误，避免后续查重/确认逻辑基于脏数据做出错误决策。
func cleanupExpiredCustomVoicePreviews() error {
	cutoff := time.Now().Add(-customVoicePreviewTTL).Unix()
	if _, err := model.DeleteExpiredMiniMaxVoicePreviews(cutoff); err != nil {
		return errors.New("音色校验失败，请稍后重试")
	}
	return nil
}

// getCustomVoiceGroupAndBilling 返回定制音色分组与扣费模型 ID。
func getCustomVoiceGroupAndBilling() (string, string) {
	group, billing := model_setting.GetCustomVoiceConfig()
	return strings.TrimSpace(group), strings.TrimSpace(billing)
}

// isDuplicateKeyErr 判断是否为唯一键冲突错误（跨数据库兼容）。
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint")
}
