package minimax

import (
	"github.com/zhongruan0522/new-api/setting/model_setting"
)

// applyModelRedirect 按 TTS 模型重定向表查找。
// 返回映射后的模型名；未命中返回原值。
func applyModelRedirect(originModel string, cfg *model_setting.MiniMaxSettings) string {
	return model_setting.ApplyMiniMaxModelRedirect(originModel, cfg)
}

// applyVoiceRedirect 按音色重定向表查找 voice_id。
// 返回映射后的 voice_id；未命中返回原值。
func applyVoiceRedirect(voice string, cfg *model_setting.MiniMaxSettings) string {
	return model_setting.ApplyMiniMaxVoiceRedirect(voice, cfg)
}

// extractEmotion 用情绪正则识别文本中的情绪标签。
// 返回：映射后的 emotion 值（未命中映射或无标签时为空串）、剥离标签后的文本。
// 行为：找到第一个匹配，提取括号内文本查 EmotionRedirect 映射；
// 无论是否命中映射，都从文本中剥离所有匹配的情绪标签。
// 正则编译失败时跳过增强（返回原文本和空 emotion），不在请求路径上崩溃——
// 正则错误属于管理员配置问题，不应阻断用户请求。
func extractEmotion(text string, pattern string, redirect map[string]string) (emotion string, cleaned string) {
	return model_setting.ExtractMiniMaxEmotion(text, pattern, redirect)
}

// replaceToneWords 用语气词正则识别文本中的语气词标签，原地替换括号内文本。
// 括号位置不变，只替换内容。映射表无此 key 时保留原标签。
// 正则编译失败时跳过增强（返回原文本）。
func replaceToneWords(text string, pattern string, redirect map[string]string) string {
	return model_setting.ReplaceMiniMaxToneWords(text, pattern, redirect)
}

// extractParenContent 从 "(content)" 格式的字符串提取 content。
// 无括号或空内容返回空串。
func extractParenContent(s string) string {
	return model_setting.ExtractMiniMaxParenContent(s)
}
