package llm

import (
	"context"
)

type ChatMessage struct {
	Role      string
	Content   string
	Thinking  string
	ToolCalls []ToolCall
}

type Tool struct {
	Type     string
	Function Function
}

type Function struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

type ToolCall struct {
	Function ToolCallFunction
}

type ToolCallFunction struct {
	Name      string
	Arguments map[string]interface{}
}

type ChatResult struct {
	Content    string
	ToolCalls  []ToolCall
	DoneReason string
}

type StageEvent struct {
	Stage   string
	Tool    string
	Message string
}

type StreamChunk struct {
	Content    string
	Thinking   string
	ToolCalls  []ToolCall
	Stage      *StageEvent
	Done       bool
	DoneReason string
	Err        error
}

type SimplyChatOptions struct {
	NumPredict  int
	NumCtx      int
	Temperature float64
}

type Client interface {
	Chat(ctx context.Context, messages []ChatMessage) (string, error)
	ChatJSON(ctx context.Context, messages []ChatMessage) (string, error)
	ChatWithTools(ctx context.Context, messages []ChatMessage, format string) (*ChatResult, error)
	ChatWithToolsDirect(ctx context.Context, messages []ChatMessage, format string, tools []Tool) (*ChatResult, error)
	ChatStream(ctx context.Context, messages []ChatMessage, tools []Tool) (<-chan StreamChunk, error)
	ChatStreamWithThink(ctx context.Context, messages []ChatMessage, tools []Tool, think bool) (<-chan StreamChunk, error)
	Schemas() []Tool
	NumCtx() int
	NumPredict() int
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	SimplyChat(ctx context.Context, prompt string) (string, error)
	SimplyChatWithOptions(ctx context.Context, prompt string, opts SimplyChatOptions) (string, error)
	GenerateChatTitle(ctx context.Context, message string) (string, error)
}

type chatRequest struct {
	Model     string         `json:"model"`
	Messages  []ChatMessage  `json:"messages"`
	Stream    bool           `json:"stream"`
	Format    string         `json:"format,omitempty"`
	Think     bool 			 `json:"think"`
	Tools     []Tool         `json:"tools,omitempty"`
	Options   map[string]any `json:"options,omitempty"`
	KeepAlive string         `json:"keep_alive,omitempty"`
}

type chatResponse struct {
	Message    ChatMessage `json:"message"`
	Done       bool        `json:"done"`
	DoneReason string      `json:"done_reason,omitempty"`
}

type embeddingRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	KeepAlive string `json:"keep_alive,omitempty"`
}

type embeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}
