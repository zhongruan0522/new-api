package controller

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"
    "sync"
    "time"

    "one-api/common"
    "one-api/dto"
    "one-api/model"
    "one-api/setting/ratio_setting"

    "github.com/gin-gonic/gin"
)

const (
    defaultTimeoutSeconds  = 10
    defaultEndpoint        = "/api/ratio_config"
    maxConcurrentFetches   = 8
)

var ratioTypes = []string{"model_ratio", "completion_ratio", "cache_ratio", "model_price"}

type upstreamResult struct {
    Name string                 `json:"name"`
    Data map[string]any         `json:"data,omitempty"`
    Err  string                 `json:"err,omitempty"`
}

func FetchUpstreamRatios(c *gin.Context) {
    var req dto.UpstreamRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
        return
    }

    if req.Timeout <= 0 {
        req.Timeout = defaultTimeoutSeconds
    }

    var upstreams []dto.UpstreamDTO

    if len(req.ChannelIDs) > 0 {
        intIds := make([]int, 0, len(req.ChannelIDs))
        for _, id64 := range req.ChannelIDs {
            intIds = append(intIds, int(id64))
        }
        dbChannels, err := model.GetChannelsByIds(intIds)
        if err != nil {
            common.LogError(c.Request.Context(), "failed to query channels: "+err.Error())
            c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询渠道失败"})
            return
        }
        for _, ch := range dbChannels {
            if base := ch.GetBaseURL(); strings.HasPrefix(base, "http") {
                upstreams = append(upstreams, dto.UpstreamDTO{
                    Name:     ch.Name,
                    BaseURL:  strings.TrimRight(base, "/"),
                    Endpoint: "",
                })
            }
        }
    }

    if len(upstreams) == 0 {
        c.JSON(http.StatusOK, gin.H{"success": false, "message": "无有效上游渠道"})
        return
    }

    var wg sync.WaitGroup
    ch := make(chan upstreamResult, len(upstreams))

    sem := make(chan struct{}, maxConcurrentFetches)

    client := &http.Client{Transport: &http.Transport{MaxIdleConns: 100, IdleConnTimeout: 90 * time.Second, TLSHandshakeTimeout: 10 * time.Second, ExpectContinueTimeout: 1 * time.Second}}

    for _, chn := range upstreams {
        wg.Add(1)
        go func(chItem dto.UpstreamDTO) {
            defer wg.Done()

            sem <- struct{}{}
            defer func() { <-sem }()

            endpoint := chItem.Endpoint
            if endpoint == "" {
                endpoint = defaultEndpoint
            } else if !strings.HasPrefix(endpoint, "/") {
                endpoint = "/" + endpoint
            }
            fullURL := chItem.BaseURL + endpoint

            ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(req.Timeout)*time.Second)
            defer cancel()

            httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
            if err != nil {
                common.LogWarn(c.Request.Context(), "build request failed: "+err.Error())
                ch <- upstreamResult{Name: chItem.Name, Err: err.Error()}
                return
            }

            resp, err := client.Do(httpReq)
            if err != nil {
                common.LogWarn(c.Request.Context(), "http error on "+chItem.Name+": "+err.Error())
                ch <- upstreamResult{Name: chItem.Name, Err: err.Error()}
                return
            }
            defer resp.Body.Close()
            if resp.StatusCode != http.StatusOK {
                common.LogWarn(c.Request.Context(), "non-200 from "+chItem.Name+": "+resp.Status)
                ch <- upstreamResult{Name: chItem.Name, Err: resp.Status}
                return
            }
            var body struct {
                Success bool                   `json:"success"`
                Data    map[string]any         `json:"data"`
                Message string                 `json:"message"`
            }
            if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
                common.LogWarn(c.Request.Context(), "json decode failed from "+chItem.Name+": "+err.Error())
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

    differences := buildDifferences(localData, successfulChannels)

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "differences":  differences,
            "test_results": testResults,
        },
    })
}

func buildDifferences(localData map[string]any, successfulChannels []struct {
    name string
    data map[string]any
}) map[string]map[string]dto.DifferenceItem {
    differences := make(map[string]map[string]dto.DifferenceItem)

    allModels := make(map[string]struct{})
    
    for _, ratioType := range ratioTypes {
        if localRatioAny, ok := localData[ratioType]; ok {
            if localRatio, ok := localRatioAny.(map[string]float64); ok {
                for modelName := range localRatio {
                    allModels[modelName] = struct{}{}
                }
            }
        }
    }
    
    for _, channel := range successfulChannels {
        for _, ratioType := range ratioTypes {
            if upstreamRatio, ok := channel.data[ratioType].(map[string]any); ok {
                for modelName := range upstreamRatio {
                    allModels[modelName] = struct{}{}
                }
            }
        }
    }

    for modelName := range allModels {
        for _, ratioType := range ratioTypes {
            var localValue interface{} = nil
            if localRatioAny, ok := localData[ratioType]; ok {
                if localRatio, ok := localRatioAny.(map[string]float64); ok {
                    if val, exists := localRatio[modelName]; exists {
                        localValue = val
                    }
                }
            }

            upstreamValues := make(map[string]interface{})
            hasUpstreamValue := false
            hasDifference := false

            for _, channel := range successfulChannels {
                var upstreamValue interface{} = nil
                
                if upstreamRatio, ok := channel.data[ratioType].(map[string]any); ok {
                    if val, exists := upstreamRatio[modelName]; exists {
                        upstreamValue = val
                        hasUpstreamValue = true
                        
                        if localValue != nil && localValue != val {
                            hasDifference = true
                        } else if localValue == val {
                            upstreamValue = "same"
                        }
                    }
                }
                if upstreamValue == nil && localValue == nil {
                    upstreamValue = "same"
                }
                
                if localValue == nil && upstreamValue != nil && upstreamValue != "same" {
                    hasDifference = true
                }
                
                upstreamValues[channel.name] = upstreamValue
            }

            shouldInclude := false
            
            if localValue != nil {
                if hasDifference {
                    shouldInclude = true
                }
            } else {
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

    channelHasDiff := make(map[string]bool)
    for _, ratioMap := range differences {
        for _, item := range ratioMap {
            for chName, val := range item.Upstreams {
                if val != nil && val != "same" {
                    channelHasDiff[chName] = true
                }
            }
        }
    }

    for modelName, ratioMap := range differences {
        for ratioType, item := range ratioMap {
            for chName := range item.Upstreams {
                if !channelHasDiff[chName] {
                    delete(item.Upstreams, chName)
                }
            }
            differences[modelName][ratioType] = item
        }
    }

    return differences
}

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