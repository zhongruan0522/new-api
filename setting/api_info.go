package setting

import (
	"encoding/json"
	"fmt"
	"net/url"
	"one-api/common"
	"regexp"
	"strings"
)

// ValidateApiInfo 验证API信息格式
func ValidateApiInfo(apiInfoStr string) error {
	if apiInfoStr == "" {
		return nil // 空字符串是合法的
	}
	
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
	
	// URL正则表达式
	urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*(/.*)?$`)
	
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