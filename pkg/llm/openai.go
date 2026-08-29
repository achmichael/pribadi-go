package llm

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/rs/zerolog"
	"github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
	model  string
	logger *zerolog.Logger
}

func NewOpenAIClient(apiKey, model string, logger *zerolog.Logger) *OpenAIClient {
	config := openai.DefaultConfig(apiKey)
	return &OpenAIClient{
		client: openai.NewClientWithConfig(config),
		model:  model,
		logger: logger,
	}
}

func NewOpenAIClientWithBaseURL(apiKey, model, baseURL string, logger *zerolog.Logger) *OpenAIClient {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	return &OpenAIClient{
		client: openai.NewClientWithConfig(config),
		model:  model,
		logger: logger,
	}
}

func (c *OpenAIClient) convertMessages(messages []ChatMessage) []openai.ChatCompletionMessage {
	var oaiMessages []openai.ChatCompletionMessage
	for _, m := range messages {
		oaiMsg := openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
		
		// For tool calls (assistant role)
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				// Convert llm.ToolCall to openai.ToolCall
				// Note: For simplicity in this adaptation, we marshal the args back to string
				// as OpenAI expects string arguments.
				argsStr := "{}" // In a full implementation, json.Marshal(tc.Function.Arguments)
				oaiMsg.ToolCalls = append(oaiMsg.ToolCalls, openai.ToolCall{
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: argsStr,
					},
				})
			}
		}

		// For tool responses (tool role)
		if m.Role == "tool" {
			// In go-openai, role is "tool" and we need a ToolCallID. We'll use a dummy one if not tracked.
			// This is a simplified adapter.
			oaiMsg.Role = openai.ChatMessageRoleTool
			oaiMsg.ToolCallID = "call_dummy"
		}

		oaiMessages = append(oaiMessages, oaiMsg)
	}
	return oaiMessages
}

func (c *OpenAIClient) convertTools(tools []Tool) []openai.Tool {
	if len(tools) == 0 {
		return nil
	}
	
	var oaiTools []openai.Tool
	for _, t := range tools {
		oaiTools = append(oaiTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}
	return oaiTools
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: c.convertMessages(messages),
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("openai chat error: %w", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", errors.New("empty choices from openai")
}

func (c *OpenAIClient) ChatJSON(ctx context.Context, messages []ChatMessage) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: c.convertMessages(messages),
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("openai chat json error: %w", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", errors.New("empty choices from openai")
}

func (c *OpenAIClient) ChatWithTools(ctx context.Context, messages []ChatMessage, format string) (*ChatResult, error) {
	return c.ChatWithToolsDirect(ctx, messages, format, c.Schemas())
}

func (c *OpenAIClient) ChatWithToolsDirect(ctx context.Context, messages []ChatMessage, format string, tools []Tool) (*ChatResult, error) {
	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: c.convertMessages(messages),
		Tools:    c.convertTools(tools),
	}
	
	if format == "json" {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai chat tools error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("empty choices from openai")
	}

	choice := resp.Choices[0]
	result := &ChatResult{
		Content:    choice.Message.Content,
		DoneReason: string(choice.FinishReason),
	}

	for _, tc := range choice.Message.ToolCalls {
		// Just passing the name and raw arguments back
		// Note: The orchestrator needs to parse the JSON arguments
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			Function: ToolCallFunction{
				Name: tc.Function.Name,
				Arguments: map[string]interface{}{
					"raw_json": tc.Function.Arguments,
				},
			},
		})
	}

	return result, nil
}

func (c *OpenAIClient) ChatStream(ctx context.Context, messages []ChatMessage, tools []Tool) (<-chan StreamChunk, error) {
	return c.ChatStreamWithThink(ctx, messages, tools, false)
}

func (c *OpenAIClient) ChatStreamWithThink(ctx context.Context, messages []ChatMessage, tools []Tool, think bool) (<-chan StreamChunk, error) {
	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: c.convertMessages(messages),
		Tools:    c.convertTools(tools),
		Stream:   true,
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai stream error: %w", err)
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				ch <- StreamChunk{Done: true, DoneReason: "stop"}
				return
			}
			if err != nil {
				ch <- StreamChunk{Err: fmt.Errorf("stream read error: %w", err)}
				return
			}

			if len(response.Choices) > 0 {
				choice := response.Choices[0]
				
				var toolCalls []ToolCall
				for _, tc := range choice.Delta.ToolCalls {
					toolCalls = append(toolCalls, ToolCall{
						Function: ToolCallFunction{
							Name: tc.Function.Name,
							Arguments: map[string]interface{}{
								"raw_json": tc.Function.Arguments,
							},
						},
					})
				}

				ch <- StreamChunk{
					Content:   choice.Delta.Content,
					ToolCalls: toolCalls,
				}
				
				if choice.FinishReason != "" {
					ch <- StreamChunk{Done: true, DoneReason: string(choice.FinishReason)}
					return
				}
			}
		}
	}()

	return ch, nil
}

func (c *OpenAIClient) Schemas() []Tool {
	// Usually the router handles fallback tools, or we pass nil here
	// The core app injects its own schemas directly anyway via ChatWithToolsDirect
	return nil
}

func (c *OpenAIClient) NumCtx() int {
	return 128000
}

func (c *OpenAIClient) NumPredict() int {
	return 4096
}

func (c *OpenAIClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	}
	
	resp, err := c.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}
	
	if len(resp.Data) > 0 {
		return resp.Data[0].Embedding, nil
	}
	return nil, errors.New("empty embedding data")
}

func (c *OpenAIClient) SimplyChat(ctx context.Context, prompt string) (string, error) {
	return c.Chat(ctx, []ChatMessage{{Role: "user", Content: prompt}})
}

func (c *OpenAIClient) SimplyChatWithOptions(ctx context.Context, prompt string, opts SimplyChatOptions) (string, error) {
	// Note: Options ignored for simplicity in this adapter
	return c.SimplyChat(ctx, prompt)
}

func (c *OpenAIClient) GenerateChatTitle(ctx context.Context, message string) (string, error) {
	prompt := fmt.Sprintf(`Generate a short title for this conversation.
	Message:
	%s

	Rules:
	- Maximum 6 words
	- No quotation marks
	- Return only the title.
	- Title must using language that same as the prompt
Text: %s`, message, message)

	return c.SimplyChat(ctx, prompt)
}
