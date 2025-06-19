package controller

import (
    "encoding/json"
    "net/http"
    "one-api/model"
    "one-api/setting/ratio_setting"
    "one-api/dto"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
)

type upstreamResult struct {
    Name string                 `json:"name"`
    Data map[string]any         `json:"data,omitempty"`
    Err  string                 `json:"err,omitempty"`
}

type TestResult struct {
    Name   string `json:"name"`
    Status string `json:"status"`
    Error  string `json:"error,omitempty"`
}

type DifferenceItem struct {
    Current   interface{}            `json:"current"`   // 当前本地值，可能为null
    Upstreams map[string]interface{} `json:"upstreams"` // 上游值：具体值/"same"/null
}

// SyncableChannel 可同步的渠道信息
type SyncableChannel struct {
    ID      int    `json:"id"`
    Name    string `json:"name"`
    BaseURL string `json:"base_url"`
    Status  int    `json:"status"`
}

// FetchUpstreamRatios 后端并发拉取上游倍率
func FetchUpstreamRatios(c *gin.Context) {
    var req dto.UpstreamRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
        return
    }

    if req.Timeout <= 0 {
        req.Timeout = 10
    }

    // build upstream list from ids + custom
    var upstreams []dto.UpstreamDTO
    if len(req.ChannelIDs) > 0 {
        // convert []int64 -> []int for model function
        intIds := make([]int, 0, len(req.ChannelIDs))
        for _, id64 := range req.ChannelIDs {
            intIds = append(intIds, int(id64))
        }
        dbChannels, _ := model.GetChannelsByIds(intIds)
        for _, ch := range dbChannels {
            upstreams = append(upstreams, dto.UpstreamDTO{
                Name:     ch.Name,
                BaseURL:  ch.GetBaseURL(),
                Endpoint: "", // assume default endpoint
            })
        }
    }
    upstreams = append(upstreams, req.CustomChannels...)

    var wg sync.WaitGroup
    ch := make(chan upstreamResult, len(upstreams))

    for _, chn := range upstreams {
        wg.Add(1)
        go func(chItem dto.UpstreamDTO) {
            defer wg.Done()
            endpoint := chItem.Endpoint
            if endpoint == "" {
                endpoint = "/api/ratio_config"
            }
            url := chItem.BaseURL + endpoint
            client := http.Client{Timeout: time.Duration(req.Timeout) * time.Second}
            resp, err := client.Get(url)
            if err != nil {
                ch <- upstreamResult{Name: chItem.Name, Err: err.Error()}
                return
            }
            defer resp.Body.Close()
            if resp.StatusCode != http.StatusOK {
                ch <- upstreamResult{Name: chItem.Name, Err: resp.Status}
                return
            }
            var body struct {
                Success bool                   `json:"success"`
                Data    map[string]any         `json:"data"`
                Message string                 `json:"message"`
            }
            if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
                ch <- upstreamResult{Name: chItem.Name, Err: err.Error()}
                return
            }
            if !body.Success {
                ch <- upstreamResult{Name: chItem.Name, Err: body.Message}
                return
            }
            ch <- upstreamResult{Name: chItem.Name, Data: body.Data}
        }(chn)
    }

    wg.Wait()
    close(ch)

    // 本地倍率配置
    localData := ratio_setting.GetExposedData()

    var testResults []dto.TestResult
    var successfulChannels []struct {
        name string
        data map[string]any
    }

    for r := range ch {
        if r.Err != "" {
            testResults = append(testResults, dto.TestResult{
                Name:   r.Name,
                Status: "error",
                Error:  r.Err,
            })
        } else {
            testResults = append(testResults, dto.TestResult{
                Name:   r.Name,
                Status: "success",
            })
            successfulChannels = append(successfulChannels, struct {
                name string
                data map[string]any
            }{name: r.Name, data: r.Data})
        }
    }

    // 构建差异化数据
    differences := buildDifferences(localData, successfulChannels)

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "differences":  differences,
            "test_results": testResults,
        },
    })
}

// buildDifferences 构建差异化数据，只返回有意义的差异
func buildDifferences(localData map[string]any, successfulChannels []struct {
    name string
    data map[string]any
}) map[string]map[string]dto.DifferenceItem {
    differences := make(map[string]map[string]dto.DifferenceItem)
    ratioTypes := []string{"model_ratio", "completion_ratio", "cache_ratio", "model_price"}

    // 收集所有模型名称
    allModels := make(map[string]struct{})
    
    // 从本地数据收集模型名称
    for _, ratioType := range ratioTypes {
        if localRatioAny, ok := localData[ratioType]; ok {
            if localRatio, ok := localRatioAny.(map[string]float64); ok {
                for modelName := range localRatio {
                    allModels[modelName] = struct{}{}
                }
            }
        }
    }
    
    // 从上游数据收集模型名称
    for _, channel := range successfulChannels {
        for _, ratioType := range ratioTypes {
            if upstreamRatio, ok := channel.data[ratioType].(map[string]any); ok {
                for modelName := range upstreamRatio {
                    allModels[modelName] = struct{}{}
                }
            }
        }
    }

    // 对每个模型和每个比率类型进行分析
    for modelName := range allModels {
        for _, ratioType := range ratioTypes {
            // 获取本地值
            var localValue interface{} = nil
            if localRatioAny, ok := localData[ratioType]; ok {
                if localRatio, ok := localRatioAny.(map[string]float64); ok {
                    if val, exists := localRatio[modelName]; exists {
                        localValue = val
                    }
                }
            }

            // 收集上游值
            upstreamValues := make(map[string]interface{})
            hasUpstreamValue := false
            hasDifference := false

            for _, channel := range successfulChannels {
                var upstreamValue interface{} = nil
                
                if upstreamRatio, ok := channel.data[ratioType].(map[string]any); ok {
                    if val, exists := upstreamRatio[modelName]; exists {
                        upstreamValue = val
                        hasUpstreamValue = true
                        
                        // 检查是否与本地值不同
                        if localValue != nil && localValue != val {
                            hasDifference = true
                        } else if localValue == val {
                            upstreamValue = "same"
                        }
                    }
                }
                
                // 如果本地值为空但上游有值，这也是差异
                if localValue == nil && upstreamValue != nil && upstreamValue != "same" {
                    hasDifference = true
                }
                
                upstreamValues[channel.name] = upstreamValue
            }

            // 应用过滤逻辑
            shouldInclude := false
            
            if localValue != nil {
                // 规则1: 本地值存在，至少有一个上游与本地值不同
                if hasDifference {
                    shouldInclude = true
                }
                // 规则2: 本地值存在，但所有上游都未设置 - 不包含
            } else {
                // 规则3: 本地值不存在，至少有一个上游设置了值
                if hasUpstreamValue {
                    shouldInclude = true
                }
            }

            if shouldInclude {
                if differences[modelName] == nil {
                    differences[modelName] = make(map[string]dto.DifferenceItem)
                }
                differences[modelName][ratioType] = dto.DifferenceItem{
                    Current:   localValue,
                    Upstreams: upstreamValues,
                }
            }
        }
    }

    return differences
}

// GetSyncableChannels 获取可用于倍率同步的渠道（base_url 不为空的渠道）
func GetSyncableChannels(c *gin.Context) {
    channels, err := model.GetAllChannels(0, 0, true, false)
    if err != nil {
        c.JSON(http.StatusOK, gin.H{
            "success": false,
            "message": err.Error(),
        })
        return
    }

    var syncableChannels []dto.SyncableChannel
    for _, channel := range channels {
        // 只返回 base_url 不为空的渠道
        if channel.GetBaseURL() != "" {
            syncableChannels = append(syncableChannels, dto.SyncableChannel{
                ID:      channel.Id,
                Name:    channel.Name,
                BaseURL: channel.GetBaseURL(),
                Status:  channel.Status,
            })
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "",
        "data":    syncableChannels,
    })
} 