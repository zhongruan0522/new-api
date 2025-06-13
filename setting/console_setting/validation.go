package console_setting

import (
    "encoding/json"
    "fmt"
    "net/url"
    "regexp"
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

    // URL 正则，支持域名 / IP
    urlRegex := regexp.MustCompile(`^https?://(?:(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?|(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))(?:\:[0-9]{1,5})?(?:/.*)?$`)

    for i, apiInfo := range apiInfoList {
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
        if !urlRegex.MatchString(urlStr) {
            return fmt.Errorf("第%d个API信息的URL格式不正确", i+1)
        }
        if _, err := url.Parse(urlStr); err != nil {
            return fmt.Errorf("第%d个API信息的URL无法解析：%s", i+1, err.Error())
        }
        if len(urlStr) > 500 {
            return fmt.Errorf("第%d个API信息的URL长度不能超过500字符", i+1)
        }
        if len(route) > 100 {
            return fmt.Errorf("第%d个API信息的线路描述长度不能超过100字符", i+1)
        }
        if len(description) > 200 {
            return fmt.Errorf("第%d个API信息的说明长度不能超过200字符", i+1)
        }
        if !validColors[color] {
            return fmt.Errorf("第%d个API信息的颜色值不合法", i+1)
        }
        dangerousChars := []string{"<script", "<iframe", "javascript:", "onload=", "onerror=", "onclick="}
        for _, d := range dangerousChars {
            lower := strings.ToLower(description)
            if strings.Contains(lower, d) || strings.Contains(strings.ToLower(route), d) {
                return fmt.Errorf("第%d个API信息包含不允许的内容", i+1)
            }
        }
    }
    return nil
}

// ValidateApiInfo 保持向后兼容
func ValidateApiInfo(apiInfoStr string) error {
    return validateApiInfo(apiInfoStr)
}

// GetApiInfo 获取 API 信息列表
func GetApiInfo() []map[string]interface{} {
    apiInfoStr := GetConsoleSetting().ApiInfo
    if apiInfoStr == "" {
        return []map[string]interface{}{}
    }
    var apiInfo []map[string]interface{}
    if err := json.Unmarshal([]byte(apiInfoStr), &apiInfo); err != nil {
        return []map[string]interface{}{}
    }
    return apiInfo
}

// ----------------- 公告 / FAQ -----------------

func validateAnnouncements(announcementsStr string) error {
    var list []map[string]interface{}
    if err := json.Unmarshal([]byte(announcementsStr), &list); err != nil {
        return fmt.Errorf("系统公告格式错误：%s", err.Error())
    }
    if len(list) > 100 {
        return fmt.Errorf("系统公告数量不能超过100个")
    }
    validTypes := map[string]bool{
        "default": true, "ongoing": true, "success": true, "warning": true, "error": true,
    }
    for i, ann := range list {
        content, ok := ann["content"].(string)
        if !ok || content == "" {
            return fmt.Errorf("第%d个公告缺少内容字段", i+1)
        }
        publishDateAny, exists := ann["publishDate"]
        if !exists {
            return fmt.Errorf("第%d个公告缺少发布日期字段", i+1)
        }
        publishDateStr, ok := publishDateAny.(string)
        if !ok || publishDateStr == "" {
            return fmt.Errorf("第%d个公告的发布日期不能为空", i+1)
        }
        if _, err := time.Parse(time.RFC3339, publishDateStr); err != nil {
            return fmt.Errorf("第%d个公告的发布日期格式错误", i+1)
        }
        if t, exists := ann["type"]; exists {
            if typeStr, ok := t.(string); ok {
                if !validTypes[typeStr] {
                    return fmt.Errorf("第%d个公告的类型值不合法", i+1)
                }
            }
        }
        if len(content) > 500 {
            return fmt.Errorf("第%d个公告的内容长度不能超过500字符", i+1)
        }
        if extra, exists := ann["extra"]; exists {
            if extraStr, ok := extra.(string); ok && len(extraStr) > 200 {
                return fmt.Errorf("第%d个公告的说明长度不能超过200字符", i+1)
            }
        }
    }
    return nil
}

func validateFAQ(faqStr string) error {
    var list []map[string]interface{}
    if err := json.Unmarshal([]byte(faqStr), &list); err != nil {
        return fmt.Errorf("FAQ信息格式错误：%s", err.Error())
    }
    if len(list) > 100 {
        return fmt.Errorf("FAQ数量不能超过100个")
    }
    for i, faq := range list {
        question, ok := faq["question"].(string)
        if !ok || question == "" {
            return fmt.Errorf("第%d个FAQ缺少问题字段", i+1)
        }
        answer, ok := faq["answer"].(string)
        if !ok || answer == "" {
            return fmt.Errorf("第%d个FAQ缺少答案字段", i+1)
        }
        if len(question) > 200 {
            return fmt.Errorf("第%d个FAQ的问题长度不能超过200字符", i+1)
        }
        if len(answer) > 1000 {
            return fmt.Errorf("第%d个FAQ的答案长度不能超过1000字符", i+1)
        }
    }
    return nil
}

// GetAnnouncements 获取系统公告
func GetAnnouncements() []map[string]interface{} {
    annStr := GetConsoleSetting().Announcements
    if annStr == "" {
        return []map[string]interface{}{}
    }
    var ann []map[string]interface{}
    _ = json.Unmarshal([]byte(annStr), &ann)
    return ann
}

// GetFAQ 获取常见问题
func GetFAQ() []map[string]interface{} {
    faqStr := GetConsoleSetting().FAQ
    if faqStr == "" {
        return []map[string]interface{}{}
    }
    var faq []map[string]interface{}
    _ = json.Unmarshal([]byte(faqStr), &faq)
    return faq
} 