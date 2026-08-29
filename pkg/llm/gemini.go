package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type GeminiClient struct {
	apiKey     string
	model      string
	logger     *zerolog.Logger
	httpClient *http.Client
}

func NewGeminiClient(apiKey, model string, logger *zerolog.Logger) *GeminiClient {
	if model == "" {
		model = "gemini-1.5-flash"
	}
	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	SystemInstruction *struct {
		Parts []geminiPart `json:"parts"`
	} `json:"systemInstruction,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func (c *GeminiClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	reqBody := geminiRequest{}
	
	for _, m := range messages {
		if m.Role == "system" {
			reqBody.SystemInstruction = &struct {
				Parts []geminiPart `json:"parts"`
			}{
				Parts: []geminiPart{{Text: m.Content}},
			}
			continue
		}
		
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		
		reqBody.Contents = append(reqBody.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var gResp geminiResponse
	if err := json.Unmarshal(bodyBytes, &gResp); err != nil {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if resp.StatusCode != http.StatusOK {
		if gResp.Error != nil {
			return "", fmt.Errorf("HTTP %d: %s", gResp.Error.Code, gResp.Error.Message)
		}
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if len(gResp.Candidates) > 0 && len(gResp.Candidates[0].Content.Parts) > 0 {
		return gResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", errors.New("empty content from gemini")
}

func (c *GeminiClient) SimplyChat(ctx context.Context, prompt string) (string, error) {
	return c.Chat(ctx, []ChatMessage{{Role: "user", Content: prompt}})
}

func (c *GeminiClient) ChatJSON(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.Chat(ctx, messages)
}

func (c *GeminiClient) ChatWithTools(ctx context.Context, messages []ChatMessage, format string) (*ChatResult, error) {
	return nil, fmt.Errorf("Gemini ChatWithTools not implemented")
}

func (c *GeminiClient) ChatWithToolsDirect(ctx context.Context, messages []ChatMessage, format string, tools []Tool) (*ChatResult, error) {
	return nil, fmt.Errorf("Gemini ChatWithToolsDirect not implemented")
}

func (c *GeminiClient) ChatStream(ctx context.Context, messages []ChatMessage, tools []Tool) (<-chan StreamChunk, error) {
	return nil, fmt.Errorf("Gemini ChatStream not implemented")
}

func (c *GeminiClient) ChatStreamWithThink(ctx context.Context, messages []ChatMessage, tools []Tool, think bool) (<-chan StreamChunk, error) {
	return nil, fmt.Errorf("Gemini ChatStreamWithThink not implemented")
}

func (c *GeminiClient) Schemas() []Tool { return nil }
func (c *GeminiClient) NumCtx() int { return 1000000 }
func (c *GeminiClient) NumPredict() int { return 8192 }
func (c *GeminiClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("Gemini Embedding not implemented")
}
func (c *GeminiClient) SimplyChatWithOptions(ctx context.Context, prompt string, opts SimplyChatOptions) (string, error) {
	return c.SimplyChat(ctx, prompt)
}
func (c *GeminiClient) GenerateChatTitle(ctx context.Context, message string) (string, error) {
	return c.SimplyChat(ctx, message)
}
