package common

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"strings"
)

type ConditionOperation struct {
	Path  string `json:"path"`  // JSON路径
	Mode  string `json:"mode"`  // full, prefix, suffix, contains
	Value string `json:"value"` // 匹配的值
}

type ParamOperation struct {
	Path       string               `json:"path"`
	Mode       string               `json:"mode"` // delete, set, move, prepend, append
	Value      interface{}          `json:"value"`
	KeepOrigin bool                 `json:"keep_origin"`
	From       string               `json:"from,omitempty"`
	To         string               `json:"to,omitempty"`
	Conditions []ConditionOperation `json:"conditions,omitempty"` // 条件列表
	Logic      string               `json:"logic,omitempty"`      // AND, OR (默认OR)
}

func ApplyParamOverride(jsonData []byte, paramOverride map[string]interface{}) ([]byte, error) {
	if len(paramOverride) == 0 {
		return jsonData, nil
	}

	// 尝试断言为操作格式
	if operations, ok := tryParseOperations(paramOverride); ok {
		// 使用新方法
		result, err := applyOperations(string(jsonData), operations)
		if err != nil {
			// 新方法失败，回退到旧方法
			return applyOperationsLegacy(jsonData, paramOverride)
		}
		return []byte(result), nil
	}

	// 直接使用旧方法
	return applyOperationsLegacy(jsonData, paramOverride)
}

func tryParseOperations(paramOverride map[string]interface{}) ([]ParamOperation, bool) {
	// 检查是否包含 "operations" 字段
	if opsValue, exists := paramOverride["operations"]; exists {
		if opsSlice, ok := opsValue.([]interface{}); ok {
			var operations []ParamOperation
			for _, op := range opsSlice {
				if opMap, ok := op.(map[string]interface{}); ok {
					operation := ParamOperation{}

					// 断言必要字段
					if path, ok := opMap["path"].(string); ok {
						operation.Path = path
					}
					if mode, ok := opMap["mode"].(string); ok {
						operation.Mode = mode
					} else {
						return nil, false // mode 是必需的
					}

					// 可选字段
					if value, exists := opMap["value"]; exists {
						operation.Value = value
					}
					if keepOrigin, ok := opMap["keep_origin"].(bool); ok {
						operation.KeepOrigin = keepOrigin
					}
					if from, ok := opMap["from"].(string); ok {
						operation.From = from
					}
					if to, ok := opMap["to"].(string); ok {
						operation.To = to
					}
					if logic, ok := opMap["logic"].(string); ok {
						operation.Logic = logic
					} else {
						operation.Logic = "OR" // 默认为OR
					}

					// 解析条件
					if conditions, exists := opMap["conditions"]; exists {
						if condSlice, ok := conditions.([]interface{}); ok {
							for _, cond := range condSlice {
								if condMap, ok := cond.(map[string]interface{}); ok {
									condition := ConditionOperation{}
									if path, ok := condMap["path"].(string); ok {
										condition.Path = path
									}
									if mode, ok := condMap["mode"].(string); ok {
										condition.Mode = mode
									}
									if value, ok := condMap["value"].(string); ok {
										condition.Value = value
									}
									operation.Conditions = append(operation.Conditions, condition)
								}
							}
						}
					}

					operations = append(operations, operation)
				} else {
					return nil, false
				}
			}
			return operations, true
		}
	}

	return nil, false
}

func checkConditions(jsonStr string, conditions []ConditionOperation, logic string) bool {
	if len(conditions) == 0 {
		return true // 没有条件，直接通过
	}
	results := make([]bool, len(conditions))

	for i, condition := range conditions {
		results[i] = checkSingleCondition(jsonStr, condition)
	}
	if strings.ToUpper(logic) == "AND" {
		for _, result := range results {
			if !result {
				return false
			}
		}
		return true
	} else {
		for _, result := range results {
			if result {
				return true
			}
		}
		return false
	}
}

func checkSingleCondition(jsonStr string, condition ConditionOperation) bool {
	value := gjson.Get(jsonStr, condition.Path)
	if !value.Exists() {
		return false
	}

	valueStr := value.String()
	targetStr := condition.Value

	switch strings.ToLower(condition.Mode) {
	case "full":
		return valueStr == targetStr
	case "prefix":
		return strings.HasPrefix(valueStr, targetStr)
	case "suffix":
		return strings.HasSuffix(valueStr, targetStr)
	case "contains":
		return strings.Contains(valueStr, targetStr)
	default:
		return valueStr == targetStr // 默认精准匹配
	}
}

// applyOperationsLegacy 原参数覆盖方法
func applyOperationsLegacy(jsonData []byte, paramOverride map[string]interface{}) ([]byte, error) {
	reqMap := make(map[string]interface{})
	err := json.Unmarshal(jsonData, &reqMap)
	if err != nil {
		return nil, err
	}

	for key, value := range paramOverride {
		reqMap[key] = value
	}

	return json.Marshal(reqMap)
}

func applyOperations(jsonStr string, operations []ParamOperation) (string, error) {
	result := jsonStr
	for _, op := range operations {
		// 检查条件是否满足
		if !checkConditions(result, op.Conditions, op.Logic) {
			continue // 条件不满足，跳过当前操作
		}

		var err error
		switch op.Mode {
		case "delete":
			result, err = sjson.Delete(result, op.Path)
		case "set":
			if op.KeepOrigin && gjson.Get(result, op.Path).Exists() {
				continue
			}
			result, err = sjson.Set(result, op.Path, op.Value)
		case "move":
			result, err = moveValue(result, op.From, op.To, op.KeepOrigin)
		case "prepend":
			result, err = modifyValue(result, op.Path, op.Value, op.KeepOrigin, true)
		case "append":
			result, err = modifyValue(result, op.Path, op.Value, op.KeepOrigin, false)
		default:
			return "", fmt.Errorf("unknown operation: %s", op.Mode)
		}
		if err != nil {
			return "", fmt.Errorf("operation %s failed: %v", op.Mode, err)
		}
	}
	return result, nil
}

func moveValue(jsonStr, fromPath, toPath string, keepOrigin bool) (string, error) {
	sourceValue := gjson.Get(jsonStr, fromPath)
	if !sourceValue.Exists() {
		return jsonStr, fmt.Errorf("source path does not exist: %s", fromPath)
	}
	if keepOrigin && gjson.Get(jsonStr, toPath).Exists() {
		return sjson.Delete(jsonStr, fromPath)
	}
	result, err := sjson.Set(jsonStr, toPath, sourceValue.Value())
	if err != nil {
		return "", err
	}
	return sjson.Delete(result, fromPath)
}

func modifyValue(jsonStr, path string, value interface{}, keepOrigin, isPrepend bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	switch {
	case current.IsArray():
		return modifyArray(jsonStr, path, value, isPrepend)
	case current.Type == gjson.String:
		return modifyString(jsonStr, path, value, isPrepend)
	case current.Type == gjson.JSON:
		return mergeObjects(jsonStr, path, value, keepOrigin)
	}
	return jsonStr, fmt.Errorf("operation not supported for type: %v", current.Type)
}

func modifyArray(jsonStr, path string, value interface{}, isPrepend bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	var newArray []interface{}
	// 添加新值
	addValue := func() {
		if arr, ok := value.([]interface{}); ok {
			newArray = append(newArray, arr...)
		} else {
			newArray = append(newArray, value)
		}
	}
	// 添加原值
	addOriginal := func() {
		current.ForEach(func(_, val gjson.Result) bool {
			newArray = append(newArray, val.Value())
			return true
		})
	}
	if isPrepend {
		addValue()
		addOriginal()
	} else {
		addOriginal()
		addValue()
	}
	return sjson.Set(jsonStr, path, newArray)
}

func modifyString(jsonStr, path string, value interface{}, isPrepend bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	valueStr := fmt.Sprintf("%v", value)
	var newStr string
	if isPrepend {
		newStr = valueStr + current.String()
	} else {
		newStr = current.String() + valueStr
	}
	return sjson.Set(jsonStr, path, newStr)
}

func mergeObjects(jsonStr, path string, value interface{}, keepOrigin bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	var currentMap, newMap map[string]interface{}

	// 解析当前值
	if err := json.Unmarshal([]byte(current.Raw), &currentMap); err != nil {
		return "", err
	}
	// 解析新值
	switch v := value.(type) {
	case map[string]interface{}:
		newMap = v
	default:
		jsonBytes, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonBytes, &newMap); err != nil {
			return "", err
		}
	}
	// 合并
	result := make(map[string]interface{})
	for k, v := range currentMap {
		result[k] = v
	}
	for k, v := range newMap {
		if !keepOrigin || result[k] == nil {
			result[k] = v
		}
	}
	return sjson.Set(jsonStr, path, result)
}
