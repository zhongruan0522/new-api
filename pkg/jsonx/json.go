package jsonx

import (
	"bytes"
	"encoding/json"
	"io"
	"unsafe"
)

func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func UnmarshalJsonStr(data string, v any) error {
	return json.Unmarshal(StringToByteSlice(data), v)
}

func DecodeJson(reader io.Reader, v any) error {
	return json.NewDecoder(reader).Decode(v)
}

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func GetJsonType(data json.RawMessage) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "unknown"
	}
	firstChar := trimmed[0]
	switch firstChar {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}

// StringToByteSlice []byte only read, panic on append
func StringToByteSlice(s string) []byte {
	tmp1 := (*[2]uintptr)(unsafe.Pointer(&s))
	tmp2 := [3]uintptr{tmp1[0], tmp1[1], tmp1[1]}
	return *(*[]byte)(unsafe.Pointer(&tmp2))
}
