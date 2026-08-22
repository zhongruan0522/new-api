package convert

import "github.com/NookMux/NookMux/internal/domain/shared"

// Converter 是 chat ↔ responses 双向协议转换的单点入口。
// 一次跨 wire 协议的请求构造一个实例：请求方向的转换会捕获 tool 代理上下文，
// 响应方向的转换消费该上下文完成 custom_tool / tool_search 的反解。
// 各 Convert* 方法只编排下方 openai_wire_convert_* 的原语函数，不携带算法；
// 流式逐帧转换因依赖方向（wire/stream → wire/convert）无法收进本类型，
// 统一经 wire/stream 的 NewChatToResponsesStreamConverter /
// NewResponsesToChatStreamConverter 两个构造器进入。
// 后续 IR 化（见 docs/PRD/prd-relay-ir-refactor.md）以同形状接口替换内部实现，
// 调用方不应绕过本类型直调包内具体转换函数。
type Converter struct {
	// Upstream / Downstream 标注本次转换的线协议方向，与
	// stream.NewStreamWriter 的同名参数含义一致。
	Upstream   shared.OpenAIWireAPI
	Downstream shared.OpenAIWireAPI

	// ToolContext 在 ConvertResponsesToChatRequest 时捕获；当请求与响应
	// 处理不在同一调用栈时（如 adaptor 转换请求、handler 处理响应），
	// 经 RelayInfo.OpenAIResponsesToolContext 传递后回填到该字段。
	ToolContext *OpenAIWireToolContext

	// ChatIncludeUsage 控制 responses → chat 流式转换是否透传 usage 帧
	//（下游 chat 请求带 stream_options.include_usage 时为 true）。
	ChatIncludeUsage bool
}

// NewConverter 创建一个 chat ↔ responses 转换会话。upstream 是上游线协议，
// downstream 是下游（客户端）线协议。
func NewConverter(upstream, downstream shared.OpenAIWireAPI) *Converter {
	return &Converter{
		Upstream:   upstream,
		Downstream: downstream,
	}
}

// ConvertChatToResponsesRequest 将下游 chat completions 请求转换为 responses 请求
// （upstream=responses 场景）。
func (cv *Converter) ConvertChatToResponsesRequest(chatReq *shared.GeneralOpenAIRequest) (*shared.OpenAIResponsesRequest, error) {
	return ConvertChatCompletionsRequestToResponsesRequest(chatReq)
}

// ConvertResponsesToChatRequest 将下游 responses 请求转换为 chat completions 请求
// （upstream=chat 场景），并把工具代理上下文捕获到 cv.ToolContext。
func (cv *Converter) ConvertResponsesToChatRequest(responsesReq *shared.OpenAIResponsesRequest) (*shared.GeneralOpenAIRequest, error) {
	chatReq, toolContext, err := ConvertResponsesRequestToChatCompletionsRequestWithToolContext(responsesReq)
	if err != nil {
		return nil, err
	}
	cv.ToolContext = toolContext
	return chatReq, nil
}

// ConvertResponsesToChatResponse 将上游 responses 非流式响应还原为 chat 响应
// （upstream=responses 场景）。上游是原生 responses，不存在工具代理，无需上下文。
func (cv *Converter) ConvertResponsesToChatResponse(responsesResp *shared.OpenAIResponsesResponse) (*shared.OpenAITextResponse, error) {
	return ConvertResponsesResponseToChatCompletionResponse(responsesResp)
}

// ConvertChatToResponsesResponse 将上游 chat 响应转换为 responses 响应
// （upstream=chat 场景），使用 cv.ToolContext 反解 custom_tool / tool_search 代理。
func (cv *Converter) ConvertChatToResponsesResponse(chatResp *shared.OpenAITextResponse) (*shared.OpenAIResponsesResponse, error) {
	return ConvertChatCompletionResponseToResponsesResponseWithToolContext(chatResp, cv.ToolContext)
}
