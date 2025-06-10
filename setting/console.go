package setting

import (
	"encoding/json"
	"fmt"
	"net/url"
	"one-api/common"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ValidateConsoleSettings 验证控制台设置信息格式
func ValidateConsoleSettings(settingsStr string, settingType string) error {
	if settingsStr == "" {
		return nil // 空字符串是合法的
	}
	
	switch settingType {
	case "ApiInfo":
		return validateApiInfo(settingsStr)
	case "Announcements":
		return validateAnnouncements(settingsStr)
	case "FAQ":
		return validateFAQ(settingsStr)
	default:
		return fmt.Errorf("未知的设置类型：%s", settingType)
	}
}

// validateApiInfo 验证API信息格式
func validateApiInfo(apiInfoStr string) error {
	var apiInfoList []map[string]interface{}
	if err := json.Unmarshal([]byte(apiInfoStr), &apiInfoList); err != nil {
		return fmt.Errorf("API信息格式错误：%s", err.Error())
	}
	
	// 验证数组长度
	if len(apiInfoList) > 50 {
		return fmt.Errorf("API信息数量不能超过50个")
	}
	
	// 允许的颜色值
	validColors := map[string]bool{
		"blue": true, "green": true, "cyan": true, "purple": true, "pink": true,
		"red": true, "orange": true, "amber": true, "yellow": true, "lime": true,
		"light-green": true, "teal": true, "light-blue": true, "indigo": true,
		"violet": true, "grey": true,
	}
	
	// URL正则表达式，支持域名和IP地址格式
	// 域名格式：https://example.com 或 https://sub.example.com:8080
	// IP地址格式：https://192.168.1.1 或 https://192.168.1.1:8080
	urlRegex := regexp.MustCompile(`^https?://(?:(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?|(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))(?::[0-9]{1,5})?(?:/.*)?$`)
	
	for i, apiInfo := range apiInfoList {
		// 检查必填字段
		urlStr, ok := apiInfo["url"].(string)
		if !ok || urlStr == "" {
			return fmt.Errorf("第%d个API信息缺少URL字段", i+1)
		}
		
		route, ok := apiInfo["route"].(string)
		if !ok || route == "" {
			return fmt.Errorf("第%d个API信息缺少线路描述字段", i+1)
		}
		
		description, ok := apiInfo["description"].(string)
		if !ok || description == "" {
			return fmt.Errorf("第%d个API信息缺少说明字段", i+1)
		}
		
		color, ok := apiInfo["color"].(string)
		if !ok || color == "" {
			return fmt.Errorf("第%d个API信息缺少颜色字段", i+1)
		}
		
		// 验证URL格式
		if !urlRegex.MatchString(urlStr) {
			return fmt.Errorf("第%d个API信息的URL格式不正确", i+1)
		}
		
		// 验证URL可解析性
		if _, err := url.Parse(urlStr); err != nil {
			return fmt.Errorf("第%d个API信息的URL无法解析：%s", i+1, err.Error())
		}
		
		// 验证字段长度
		if len(urlStr) > 500 {
			return fmt.Errorf("第%d个API信息的URL长度不能超过500字符", i+1)
		}
		
		if len(route) > 100 {
			return fmt.Errorf("第%d个API信息的线路描述长度不能超过100字符", i+1)
		}
		
		if len(description) > 200 {
			return fmt.Errorf("第%d个API信息的说明长度不能超过200字符", i+1)
		}
		
		// 验证颜色值
		if !validColors[color] {
			return fmt.Errorf("第%d个API信息的颜色值不合法", i+1)
		}
		
		// 检查并过滤危险字符（防止XSS）
		dangerousChars := []string{"<script", "<iframe", "javascript:", "onload=", "onerror=", "onclick="}
		for _, dangerous := range dangerousChars {
			if strings.Contains(strings.ToLower(description), dangerous) {
				return fmt.Errorf("第%d个API信息的说明包含不允许的内容", i+1)
			}
			if strings.Contains(strings.ToLower(route), dangerous) {
				return fmt.Errorf("第%d个API信息的线路描述包含不允许的内容", i+1)
			}
		}
	}
	
	return nil
}

// ValidateApiInfo 保持向后兼容的函数
func ValidateApiInfo(apiInfoStr string) error {
	return validateApiInfo(apiInfoStr)
}

// GetApiInfo 获取API信息列表
func GetApiInfo() []map[string]interface{} {
	// 从OptionMap中获取API信息，如果不存在则返回空数组
	common.OptionMapRWMutex.RLock()
	apiInfoStr, exists := common.OptionMap["ApiInfo"]
	common.OptionMapRWMutex.RUnlock()
	
	if !exists || apiInfoStr == "" {
		// 如果没有配置，返回空数组
		return []map[string]interface{}{}
	}
	
	// 解析存储的API信息
	var apiInfo []map[string]interface{}
	if err := json.Unmarshal([]byte(apiInfoStr), &apiInfo); err != nil {
		// 如果解析失败，返回空数组
		return []map[string]interface{}{}
	}
	
	return apiInfo
}

// validateAnnouncements 验证系统公告格式
func validateAnnouncements(announcementsStr string) error {
	var announcementsList []map[string]interface{}
	if err := json.Unmarshal([]byte(announcementsStr), &announcementsList); err != nil {
		return fmt.Errorf("系统公告格式错误：%s", err.Error())
	}
	
	// 验证数组长度
	if len(announcementsList) > 100 {
		return fmt.Errorf("系统公告数量不能超过100个")
	}
	
	// 允许的类型值
	validTypes := map[string]bool{
		"default": true, "ongoing": true, "success": true, "warning": true, "error": true,
	}
	
	for i, announcement := range announcementsList {
		// 检查必填字段
		content, ok := announcement["content"].(string)
		if !ok || content == "" {
			return fmt.Errorf("第%d个公告缺少内容字段", i+1)
		}
		
		// 检查发布日期字段
		publishDate, exists := announcement["publishDate"]
		if !exists {
			return fmt.Errorf("第%d个公告缺少发布日期字段", i+1)
		}
		
		publishDateStr, ok := publishDate.(string)
		if !ok || publishDateStr == "" {
			return fmt.Errorf("第%d个公告的发布日期不能为空", i+1)
		}
		
		// 验证ISO日期格式
		if _, err := time.Parse(time.RFC3339, publishDateStr); err != nil {
			return fmt.Errorf("第%d个公告的发布日期格式错误", i+1)
		}
		
		// 验证可选字段
		if announcementType, exists := announcement["type"]; exists {
			if typeStr, ok := announcementType.(string); ok {
				if !validTypes[typeStr] {
					return fmt.Errorf("第%d个公告的类型值不合法", i+1)
				}
			}
		}
		
		// 验证字段长度
		if len(content) > 500 {
			return fmt.Errorf("第%d个公告的内容长度不能超过500字符", i+1)
		}
		
		if extra, exists := announcement["extra"]; exists {
			if extraStr, ok := extra.(string); ok && len(extraStr) > 200 {
				return fmt.Errorf("第%d个公告的说明长度不能超过200字符", i+1)
			}
		}
		
		// 检查并过滤危险字符（防止XSS）
		dangerousChars := []string{"<script", "<iframe", "javascript:", "onload=", "onerror=", "onclick="}
		for _, dangerous := range dangerousChars {
			if strings.Contains(strings.ToLower(content), dangerous) {
				return fmt.Errorf("第%d个公告的内容包含不允许的内容", i+1)
			}
		}
	}
	
	return nil
}

// validateFAQ 验证常见问答格式
func validateFAQ(faqStr string) error {
	var faqList []map[string]interface{}
	if err := json.Unmarshal([]byte(faqStr), &faqList); err != nil {
		return fmt.Errorf("常见问答格式错误：%s", err.Error())
	}
	
	// 验证数组长度
	if len(faqList) > 100 {
		return fmt.Errorf("常见问答数量不能超过100个")
	}
	
	for i, faq := range faqList {
		// 检查必填字段
		title, ok := faq["title"].(string)
		if !ok || title == "" {
			return fmt.Errorf("第%d个问答缺少标题字段", i+1)
		}
		
		content, ok := faq["content"].(string)
		if !ok || content == "" {
			return fmt.Errorf("第%d个问答缺少内容字段", i+1)
		}
		
		// 验证字段长度
		if len(title) > 200 {
			return fmt.Errorf("第%d个问答的标题长度不能超过200字符", i+1)
		}
		
		if len(content) > 1000 {
			return fmt.Errorf("第%d个问答的内容长度不能超过1000字符", i+1)
		}
		
		// 检查并过滤危险字符（防止XSS）
		dangerousChars := []string{"<script", "<iframe", "javascript:", "onload=", "onerror=", "onclick="}
		for _, dangerous := range dangerousChars {
			if strings.Contains(strings.ToLower(title), dangerous) {
				return fmt.Errorf("第%d个问答的标题包含不允许的内容", i+1)
			}
			if strings.Contains(strings.ToLower(content), dangerous) {
				return fmt.Errorf("第%d个问答的内容包含不允许的内容", i+1)
			}
		}
	}
	
	return nil
}

// GetAnnouncements 获取系统公告列表（返回最新的前20条）
func GetAnnouncements() []map[string]interface{} {
	common.OptionMapRWMutex.RLock()
	announcementsStr, exists := common.OptionMap["Announcements"]
	common.OptionMapRWMutex.RUnlock()
	
	if !exists || announcementsStr == "" {
		return []map[string]interface{}{}
	}
	
	var announcements []map[string]interface{}
	if err := json.Unmarshal([]byte(announcementsStr), &announcements); err != nil {
		return []map[string]interface{}{}
	}
	
	// 按发布日期降序排序（最新的在前）
	sort.Slice(announcements, func(i, j int) bool {
		dateI, okI := announcements[i]["publishDate"].(string)
		dateJ, okJ := announcements[j]["publishDate"].(string)
		
		if !okI || !okJ {
			return false
		}
		
		timeI, errI := time.Parse(time.RFC3339, dateI)
		timeJ, errJ := time.Parse(time.RFC3339, dateJ)
		
		if errI != nil || errJ != nil {
			return false
		}
		
		return timeI.After(timeJ)
	})
	
	// 限制返回前20条
	if len(announcements) > 20 {
		announcements = announcements[:20]
	}
	
	return announcements
}

// GetFAQ 获取常见问答列表
func GetFAQ() []map[string]interface{} {
	common.OptionMapRWMutex.RLock()
	faqStr, exists := common.OptionMap["FAQ"]
	common.OptionMapRWMutex.RUnlock()
	
	if !exists || faqStr == "" {
		return []map[string]interface{}{}
	}
	
	var faq []map[string]interface{}
	if err := json.Unmarshal([]byte(faqStr), &faq); err != nil {
		return []map[string]interface{}{}
	}
	
	return faq
} 