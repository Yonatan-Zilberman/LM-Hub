package api

// Message represents a single chat message.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatRequest represents the payload for chat completion.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	TopP        float64   `json:"top_p"`
	Stream      bool      `json:"stream"`
}

// Delta represents the incremental change in a streaming choice.
type Delta struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Choice represents a single choice in the chat completion.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	Delta        Delta   `json:"delta"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse represents the complete response from chat completion.
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// ModelInstance represents a loaded instance of a model.
type ModelInstance struct {
	ID     string                 `json:"id"`
	Config map[string]interface{} `json:"config"`
}

// ModelInfo represents information about an available model.
type ModelInfo struct {
	Type             string          `json:"type"`
	Publisher        string          `json:"publisher"`
	Key              string          `json:"key"`
	DisplayName      string          `json:"display_name"`
	Architecture     string          `json:"architecture"`
	LoadedInstances  []ModelInstance `json:"loaded_instances"`
	MaxContextLength int             `json:"max_context_length"`
	Format           string          `json:"format"`
	SizeBytes        int64           `json:"size_bytes"`
}

// ModelListResponse is the response wrapper for /api/v1/models.
type ModelListResponse struct {
	Models []ModelInfo `json:"models"`
}

// LoadModelRequest is the payload for /api/v1/models/load.
type LoadModelRequest struct {
	Model         string `json:"model"`
	ContextLength int    `json:"context_length,omitempty"`
}

// LoadModelResponse is the response from /api/v1/models/load.
type LoadModelResponse struct {
	Type            string  `json:"type"`
	InstanceID      string  `json:"instance_id"`
	LoadTimeSeconds float64 `json:"load_time_seconds"`
	Status          string  `json:"status"`
}

// UnloadModelRequest is the payload for /api/v1/models/unload.
type UnloadModelRequest struct {
	InstanceID string `json:"instance_id"`
}

// UnloadModelResponse is the response from /api/v1/models/unload.
type UnloadModelResponse struct {
	InstanceID string `json:"instance_id"`
}

// LoadedModelInfo represents mock/actual telemetry of a loaded model.
type LoadedModelInfo struct {
	ModelID         string  `json:"model_id"`
	ContextLength   int     `json:"context_length"`
	TokensUsed      int     `json:"tokens_used"`
	TokensFree      int     `json:"tokens_free"`
	FillPercent     float64 `json:"fill_pct"`
	RAMUsedGB       float64 `json:"ram_used_gb"`
	TokensPerSecond float64 `json:"tokens_per_sec"`
	TTFT_ms         int     `json:"ttft_ms"`
}
