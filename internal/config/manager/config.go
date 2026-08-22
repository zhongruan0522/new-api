package manager

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/NookMux/NookMux/internal/common"
	"github.com/NookMux/NookMux/pkg/jsonx"
)

// ConfigManager 统一管理所有配置
type ConfigManager struct {
	configs map[string]interface{}
	mutex   sync.RWMutex
}

var GlobalConfig = NewConfigManager()

func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configs: make(map[string]interface{}),
	}
}

// Register 注册一个配置模块
func (cm *ConfigManager) Register(name string, config interface{}) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.configs[name] = config
}

// Get 获取指定配置模块
func (cm *ConfigManager) Get(name string) interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.configs[name]
}

// LoadFromDB 从数据库加载配置
func (cm *ConfigManager) LoadFromDB(options map[string]string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for name, config := range cm.configs {
		prefix := name + "."
		configMap := make(map[string]string)

		// 收集属于此配置的所有选项
		for key, value := range options {
			if strings.HasPrefix(key, prefix) {
				configKey := strings.TrimPrefix(key, prefix)
				configMap[configKey] = value
			}
		}

		// 如果找到配置项，则更新配置
		if len(configMap) > 0 {
			if err := updateConfigFromMap(config, configMap); err != nil {
				common.SysError("failed to update config " + name + ": " + err.Error())
				continue
			}
		}
	}

	return nil
}

// SaveToDB 将配置保存到数据库
func (cm *ConfigManager) SaveToDB(updateFunc func(key, value string) error) error {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	for name, config := range cm.configs {
		configMap, err := configToMap(config)
		if err != nil {
			return err
		}

		for key, value := range configMap {
			dbKey := name + "." + key
			if err := updateFunc(dbKey, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// 辅助函数：将配置对象转换为map
func configToMap(config interface{}) (map[string]string, error) {
	result := make(map[string]string)

	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过未导出字段
		if !fieldType.IsExported() {
			continue
		}

		key, ok := jsonFieldName(fieldType)
		if !ok {
			continue
		}

		// 处理不同类型的字段
		var strValue string
		switch field.Kind() {
		case reflect.String:
			strValue = field.String()
		case reflect.Bool:
			strValue = strconv.FormatBool(field.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			strValue = strconv.FormatInt(field.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			strValue = strconv.FormatUint(field.Uint(), 10)
		case reflect.Float32, reflect.Float64:
			strValue = strconv.FormatFloat(field.Float(), 'f', -1, 64)
		case reflect.Ptr:
			// 处理指针类型：如果非 nil，序列化指向的值
			if !field.IsNil() {
				bytes, err := jsonx.Marshal(field.Interface())
				if err != nil {
					return nil, err
				}
				strValue = string(bytes)
			} else {
				// nil 指针序列化为 "null"
				strValue = "null"
			}
		case reflect.Map, reflect.Slice, reflect.Struct:
			// 复杂类型使用JSON序列化
			bytes, err := jsonx.Marshal(field.Interface())
			if err != nil {
				return nil, err
			}
			strValue = string(bytes)
		default:
			// 跳过不支持的类型
			continue
		}

		result[key] = strValue
	}

	return result, nil
}

// 辅助函数：从map更新配置对象
func updateConfigFromMap(config interface{}, configMap map[string]string) error {
	val := reflect.ValueOf(config)
	if val.Kind() != reflect.Ptr {
		return nil
	}
	val = val.Elem()

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过未导出字段
		if !fieldType.IsExported() {
			continue
		}

		key, ok := jsonFieldName(fieldType)
		if !ok {
			continue
		}

		// 检查map中是否有对应的值
		strValue, ok := configMap[key]
		if !ok {
			continue
		}

		if !field.CanSet() {
			continue
		}

		if err := setFieldFromString(field, strValue); err != nil {
			return fmt.Errorf("update config field %s: %w", key, err)
		}
	}

	return nil
}

func validateConfigFromMap(config interface{}, configMap map[string]string) error {
	val := reflect.ValueOf(config)
	if val.Kind() != reflect.Ptr {
		return nil
	}
	val = val.Elem()

	if val.Kind() != reflect.Struct {
		return nil
	}

	clone := reflect.New(val.Type())
	clone.Elem().Set(val)
	return updateConfigFromMap(clone.Interface(), configMap)
}

func jsonFieldName(fieldType reflect.StructField) (string, bool) {
	tag := fieldType.Tag.Get("json")
	name := strings.Split(tag, ",")[0]
	if name == "-" {
		return "", false
	}
	if name == "" {
		name = fieldType.Name
	}
	return name, true
}

func setFieldFromString(field reflect.Value, strValue string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(strValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(strValue)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := parseIntConfigValue(strValue)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := parseUintConfigValue(strValue)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(strValue, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	case reflect.Ptr:
		if strValue == "null" {
			field.Set(reflect.Zero(field.Type()))
			return nil
		}
		newValue := reflect.New(field.Type().Elem())
		if err := jsonx.Unmarshal([]byte(strValue), newValue.Interface()); err != nil {
			return err
		}
		field.Set(newValue)
	case reflect.Map, reflect.Slice, reflect.Struct:
		newValue := reflect.New(field.Type())
		if err := jsonx.Unmarshal([]byte(strValue), newValue.Interface()); err != nil {
			// 兼容历史持久化数据中 `[]string` 字段被写成 JSON 数字数组（如
			// allowed_ports 存为 [80,443] 而非 ["80","443"]）的情形。这种情况
			// 直接 Unmarshal 进 []string 会失败，导致整个配置模块被跳过、所有
			// 字段在重启后回到默认值。此处仅对 []string 目标做一次容错：解析为
			// 通用数组后逐元素转字符串，其他 slice/map/struct 类型仍保持严格。
			if coerced, ok := coerceStringSlice(strValue, field.Type()); ok {
				field.Set(coerced)
				return nil
			}
			return err
		}
		field.Set(newValue.Elem())
	}
	return nil
}

// coerceStringSlice 尝试把 JSON 文本（可能是数字数组、混合类型数组）规范化为
// 目标 []string 切片。仅当目标是元素类型为 string 的切片时生效；成功返回可用于
// 赋值的 reflect.Value，失败返回 (zero, false)，由调用方决定如何报错。
func coerceStringSlice(strValue string, targetType reflect.Type) (reflect.Value, bool) {
	if targetType.Kind() != reflect.Slice || targetType.Elem().Kind() != reflect.String {
		return reflect.Value{}, false
	}
	var raw []interface{}
	if err := jsonx.Unmarshal([]byte(strValue), &raw); err != nil {
		return reflect.Value{}, false
	}
	out := reflect.MakeSlice(targetType, 0, len(raw))
	for _, item := range raw {
		switch v := item.(type) {
		case string:
			out = reflect.Append(out, reflect.ValueOf(v))
		case float64, int, int64, json.Number, bool:
			out = reflect.Append(out, reflect.ValueOf(fmt.Sprintf("%v", v)))
		default:
			// 含 nil、嵌套对象等无法无损转字符串的元素，放弃容错。
			return reflect.Value{}, false
		}
	}
	return out, true
}

func parseIntConfigValue(strValue string) (int64, error) {
	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err == nil {
		return intValue, nil
	}

	// 兼容 float 格式的字符串（如 "2.000000"）
	floatValue, fErr := strconv.ParseFloat(strValue, 64)
	if fErr != nil {
		return 0, err
	}
	return int64(floatValue), nil
}

func parseUintConfigValue(strValue string) (uint64, error) {
	uintValue, err := strconv.ParseUint(strValue, 10, 64)
	if err == nil {
		return uintValue, nil
	}

	// 兼容 float 格式的字符串
	floatValue, fErr := strconv.ParseFloat(strValue, 64)
	if fErr != nil || floatValue < 0 {
		return 0, err
	}
	return uint64(floatValue), nil
}

// ConfigToMap 将配置对象转换为map（导出函数）
func ConfigToMap(config interface{}) (map[string]string, error) {
	return configToMap(config)
}

// UpdateConfigFromMap 从map更新配置对象（导出函数）
func UpdateConfigFromMap(config interface{}, configMap map[string]string) error {
	return updateConfigFromMap(config, configMap)
}

// ValidateConfigFromMap validates a config update without mutating the current config object.
func ValidateConfigFromMap(config interface{}, configMap map[string]string) error {
	return validateConfigFromMap(config, configMap)
}

// ExportAllConfigs 导出所有已注册的配置为扁平结构
func (cm *ConfigManager) ExportAllConfigs() map[string]string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	result := make(map[string]string)

	for name, cfg := range cm.configs {
		configMap, err := ConfigToMap(cfg)
		if err != nil {
			continue
		}

		// 使用 "模块名.配置项" 的格式添加到结果中
		for key, value := range configMap {
			result[name+"."+key] = value
		}
	}

	return result
}
