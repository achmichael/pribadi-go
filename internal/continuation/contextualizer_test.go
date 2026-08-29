package continuation

import (
	"context"
	"testing"

	"github.com/achmichael/pribadi-go/pkg/llm"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOllamaClient is a mock for the OllamaClient interface
type MockOllamaClient struct {
	mock.Mock
}

func (m *MockOllamaClient) Chat(ctx context.Context, messages []llm.ChatMessage) (string, error) {
	args := m.Called(ctx, messages)
	return args.String(0), args.Error(1)
}

func (m *MockOllamaClient) ChatJSON(ctx context.Context, messages []llm.ChatMessage) (string, error) {
	args := m.Called(ctx, messages)
	return args.String(0), args.Error(1)
}

func (m *MockOllamaClient) ChatWithTools(ctx context.Context, messages []llm.ChatMessage, format string) (*llm.ChatResult, error) {
	args := m.Called(ctx, messages, format)
	return args.Get(0).(*llm.ChatResult), args.Error(1)
}

func (m *MockOllamaClient) ChatWithToolsDirect(ctx context.Context, messages []llm.ChatMessage, format string, tools []llm.Tool) (*llm.ChatResult, error) {
	args := m.Called(ctx, messages, format, tools)
	return args.Get(0).(*llm.ChatResult), args.Error(1)
}

func (m *MockOllamaClient) ChatStream(ctx context.Context, messages []llm.ChatMessage, tools []llm.Tool) (<-chan llm.StreamChunk, error) {
	args := m.Called(ctx, messages, tools)
	return args.Get(0).(<-chan llm.StreamChunk), args.Error(1)
}

func (m *MockOllamaClient) ChatStreamWithThink(ctx context.Context, messages []llm.ChatMessage, tools []llm.Tool, think bool) (<-chan llm.StreamChunk, error) {
	args := m.Called(ctx, messages, tools, think)
	return args.Get(0).(<-chan llm.StreamChunk), args.Error(1)
}

func (m *MockOllamaClient) Schemas() []llm.Tool {
	args := m.Called()
	return args.Get(0).([]llm.Tool)
}

func (m *MockOllamaClient) NumCtx() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockOllamaClient) NumPredict() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockOllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]float32), args.Error(1)
}

func (m *MockOllamaClient) SimplyChat(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockOllamaClient) GenerateChatTitle(ctx context.Context, message string) (string, error) {
	args := m.Called(ctx, message)
	return args.String(0), args.Error(1)
}

func TestContextualizer_Analyze_Continue(t *testing.T) {
	testCases := []struct {
		name           string
		userText       string
		history        []HistoryMessage
		ollamaResponse string
		expectedType   AnalysisType
	}{
		{
			name:     "Indonesian lanjutkan",
			userText: "lanjutkan",
			history: []HistoryMessage{
				{Role: "assistant", Content: "Dokumen ini membahas tentang machine learning. Ada beberapa konsep penting seperti supervised learning, unsupervised learning..."},
			},
			ollamaResponse: `{"type": "CONTINUE"}`,
			expectedType:   TypeContinue,
		},
		{
			name:     "Indonesian terus gimana",
			userText: "terus gimana",
			history: []HistoryMessage{
				{Role: "assistant", Content: "Untuk membuat nasi goreng, pertama siapkan bahan-bahan: nasi putih, telur, bawang merah..."},
			},
			ollamaResponse: `{"type": "CONTINUE"}`,
			expectedType:   TypeContinue,
		},
		{
			name:     "Indonesian lanjut dong",
			userText: "lanjut dong",
			history: []HistoryMessage{
				{Role: "assistant", Content: "Resep kue brownies membutuhkan coklat, telur, gula, dan tepung. Langkah pertama..."},
			},
			ollamaResponse: `{"type": "CONTINUE"}`,
			expectedType:   TypeContinue,
		},
		{
			name:     "English continue",
			userText: "continue",
			history: []HistoryMessage{
				{Role: "assistant", Content: "Python is a popular programming language for data science..."},
			},
			ollamaResponse: `{"type": "CONTINUE"}`,
			expectedType:   TypeContinue,
		},
		{
			name:     "English next",
			userText: "next",
			history: []HistoryMessage{
				{Role: "assistant", Content: "Step 1: Initialize the project..."},
			},
			ollamaResponse: `{"type": "CONTINUE"}`,
			expectedType:   TypeContinue,
		},
		{
			name:     "English more",
			userText: "more",
			history: []HistoryMessage{
				{Role: "assistant", Content: "The first chapter introduces the basics..."},
			},
			ollamaResponse: `{"type": "CONTINUE"}`,
			expectedType:   TypeContinue,
		},
		{
			name:     "English go on",
			userText: "go on",
			history: []HistoryMessage{
				{Role: "assistant", Content: "The story begins in a small village..."},
			},
			ollamaResponse: `{"type": "CONTINUE"}`,
			expectedType:   TypeContinue,
		},
		{
			name:     "Empty history still continues",
			userText: "lanjutkan",
			history:  []HistoryMessage{},
			ollamaResponse: `{"type": "CONTINUE"}`,
			expectedType:   TypeContinue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockOllama := new(MockOllamaClient)
			mockOllama.On("ChatJSON", mock.Anything, mock.Anything).Return(tc.ollamaResponse, nil)

			logger := zerolog.Nop()
			ctx := NewContextualizer(mockOllama, &logger)

			result, err := ctx.Analyze(context.Background(), tc.userText, tc.history)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedType, result.Type)
			mockOllama.AssertExpectations(t)
		})
	}
}

func TestContextualizer_Analyze_Standalone(t *testing.T) {
	testCases := []struct {
		name           string
		userText       string
		history        []HistoryMessage
		ollamaResponse string
		expectedType   AnalysisType
		expectedQuery  string
	}{
		{
			name:     "New question after answer",
			userText: "apa itu JavaScript?",
			history: []HistoryMessage{
				{Role: "assistant", Content: "Python adalah bahasa pemrograman yang populer untuk data science..."},
			},
			ollamaResponse: `{"type": "STANDALONE", "rewritten_query": "Apa itu JavaScript?"}`,
			expectedType:   TypeStandalone,
			expectedQuery:  "Apa itu JavaScript?",
		},
		{
			name:     "Follow-up question about different entity",
			userText: "berapa umurnya?",
			history: []HistoryMessage{
				{Role: "user", Content: "siapa presiden Indonesia?"},
				{Role: "assistant", Content: "Presiden Indonesia saat ini adalah Joko Widodo..."},
			},
			ollamaResponse: `{"type": "STANDALONE", "rewritten_query": "Berapa umur Presiden Indonesia saat ini?"}`,
			expectedType:   TypeStandalone,
			expectedQuery:  "Berapa umur Presiden Indonesia saat ini?",
		},
		{
			name:     "Empty history with new query",
			userText: "jelaskan tentang AI",
			history:  []HistoryMessage{},
			ollamaResponse: `{"type": "STANDALONE", "rewritten_query": "Jelaskan tentang AI"}`,
			expectedType:   TypeStandalone,
			expectedQuery:  "Jelaskan tentang AI",
		},
		{
			name:     "Completely new topic",
			userText: "cuaca hari ini gimana?",
			history: []HistoryMessage{
				{Role: "assistant", Content: "Machine learning adalah subset dari AI..."},
			},
			ollamaResponse: `{"type": "STANDALONE", "rewritten_query": "Cuaca hari ini gimana?"}`,
			expectedType:   TypeStandalone,
			expectedQuery:  "Cuaca hari ini gimana?",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockOllama := new(MockOllamaClient)
			mockOllama.On("ChatJSON", mock.Anything, mock.Anything).Return(tc.ollamaResponse, nil)

			logger := zerolog.Nop()
			ctx := NewContextualizer(mockOllama, &logger)

			result, err := ctx.Analyze(context.Background(), tc.userText, tc.history)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedType, result.Type)
			assert.Equal(t, tc.expectedQuery, result.RewrittenQuery)
			mockOllama.AssertExpectations(t)
		})
	}
}

func TestContextualizer_Analyze_OllamaError_FallbacksToStandalone(t *testing.T) {
	mockOllama := new(MockOllamaClient)
	mockOllama.On("ChatJSON", mock.Anything, mock.Anything).Return("", assert.AnError)

	logger := zerolog.Nop()
	ctx := NewContextualizer(mockOllama, &logger)

	result, err := ctx.Analyze(context.Background(), "test query", []HistoryMessage{})

	assert.NoError(t, err)
	assert.Equal(t, TypeStandalone, result.Type)
	assert.Equal(t, "test query", result.RewrittenQuery)
	mockOllama.AssertExpectations(t)
}

func TestContextualizer_Analyze_InvalidJSON_FallbacksToStandalone(t *testing.T) {
	mockOllama := new(MockOllamaClient)
	mockOllama.On("ChatJSON", mock.Anything, mock.Anything).Return("invalid json{", nil)

	logger := zerolog.Nop()
	ctx := NewContextualizer(mockOllama, &logger)

	result, err := ctx.Analyze(context.Background(), "test query", []HistoryMessage{})

	assert.NoError(t, err)
	assert.Equal(t, TypeStandalone, result.Type)
	assert.Equal(t, "test query", result.RewrittenQuery)
	mockOllama.AssertExpectations(t)
}

func TestContextualizer_Analyze_EmptyResponse_FallbacksToStandalone(t *testing.T) {
	mockOllama := new(MockOllamaClient)
	mockOllama.On("ChatJSON", mock.Anything, mock.Anything).Return("", nil)

	logger := zerolog.Nop()
	ctx := NewContextualizer(mockOllama, &logger)

	result, err := ctx.Analyze(context.Background(), "test query", []HistoryMessage{})

	assert.NoError(t, err)
	assert.Equal(t, TypeStandalone, result.Type)
	assert.Equal(t, "test query", result.RewrittenQuery)
	mockOllama.AssertExpectations(t)
}