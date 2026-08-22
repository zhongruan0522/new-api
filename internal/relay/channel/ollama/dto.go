package ollama

type OllamaOpenAIModelListResponse struct {
	Data []OllamaOpenAIModel `json:"data"`
}

type OllamaOpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object,omitempty"`
	Created int64  `json:"created,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}

type OllamaModel struct {
	Name       string            `json:"name"`
	Created    int64             `json:"created,omitempty"`
	OwnedBy    string            `json:"owned_by,omitempty"`
	Size       int64             `json:"size"`
	Digest     string            `json:"digest,omitempty"`
	ModifiedAt string            `json:"modified_at"`
	Details    OllamaModelDetail `json:"details,omitempty"`
}

type OllamaModelDetail struct {
	ParentModel       string   `json:"parent_model,omitempty"`
	Format            string   `json:"format,omitempty"`
	Family            string   `json:"family,omitempty"`
	Families          []string `json:"families,omitempty"`
	ParameterSize     string   `json:"parameter_size,omitempty"`
	QuantizationLevel string   `json:"quantization_level,omitempty"`
}

type OllamaPullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream,omitempty"`
}

type OllamaPullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

type OllamaDeleteRequest struct {
	Name string `json:"name"`
}
