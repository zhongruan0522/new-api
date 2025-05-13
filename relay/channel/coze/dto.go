package coze

import "encoding/json"

// type CozeResponse struct {
// 	Code    int                  `json:"code"`
// 	Message string               `json:"message"`
// 	Data    CozeConversationData `json:"data"`
// 	Detail  CozeConversationData `json:"detail"`
// }

// type CozeConversationData struct {
// 	Id            string          `json:"id"`
// 	CreatedAt     int64           `json:"created_at"`
// 	MetaData      json.RawMessage `json:"meta_data"`
// 	LastSectionId string          `json:"last_section_id"`
// }

// type CozeResponseDetail struct {
// 	Logid string `json:"logid"`
// }

type CozeError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// type CozeErrorWithStatusCode struct {
// 	Error      CozeError `json:"error"`
// 	StatusCode int
// 	LocalError bool
// }

type CozeRequest struct {
	BotId    string             `json:"bot_id,omitempty"`
	MetaData json.RawMessage    `json:"meta_data,omitempty"`
	Messages []CozeEnterMessage `json:"messages,omitempty"`
}

type CozeEnterMessage struct {
	Role        string          `json:"role"`
	Type        string          `json:"type,omitempty"`
	Content     json.RawMessage `json:"content,omitempty"`
	MetaData    json.RawMessage `json:"meta_data,omitempty"`
	ContentType string          `json:"content_type,omitempty"`
}

type CozeChatRequest struct {
	BotId              string             `json:"bot_id"`
	UserId             string             `json:"user_id"`
	AdditionalMessages []CozeEnterMessage `json:"additional_messages,omitempty"`
	Stream             bool               `json:"stream,omitempty"`
	CustomVariables    json.RawMessage    `json:"custom_variables,omitempty"`
	AutoSaveHistory    bool               `json:"auto_save_history,omitempty"`
	MetaData           json.RawMessage    `json:"meta_data,omitempty"`
	ExtraParams        json.RawMessage    `json:"extra_params,omitempty"`
	ShortcutCommand    json.RawMessage    `json:"shortcut_command,omitempty"`
	Parameters         json.RawMessage    `json:"parameters,omitempty"`
}

type CozeChatResponse struct {
	Code int                  `json:"code"`
	Msg  string               `json:"msg"`
	Data CozeChatResponseData `json:"data"`
}

type CozeChatResponseData struct {
	Id             string        `json:"id"`
	ConversationId string        `json:"conversation_id"`
	BotId          string        `json:"bot_id"`
	CreatedAt      int64         `json:"created_at"`
	LastError      CozeError     `json:"last_error"`
	Status         string        `json:"status"`
	Usage          CozeChatUsage `json:"usage"`
}

type CozeChatUsage struct {
	TokenCount  int `json:"token_count"`
	OutputCount int `json:"output_count"`
	InputCount  int `json:"input_count"`
}
