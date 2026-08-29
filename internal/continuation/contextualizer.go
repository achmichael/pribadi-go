package continuation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/pkg/llm"
	"github.com/rs/zerolog"
)

type AnalysisType string

const (
	TypeContinue   AnalysisType = "CONTINUE"
	TypeStandalone AnalysisType = "STANDALONE"
)

type AnalysisResult struct {
	Type           AnalysisType
	RewrittenQuery string
}

type HistoryMessage struct {
	Role    string
	Content string
}

type Contextualizer interface {
	Analyze(ctx context.Context, userText string, history []HistoryMessage) (*AnalysisResult, error)
}

type llmContextualizer struct {
	ollama llm.Client
	logger *zerolog.Logger
}

func NewContextualizer(ollama llm.Client, logger *zerolog.Logger) Contextualizer {
	return &llmContextualizer{
		ollama: ollama,
		logger: logger,
	}
}

const contextualizationPrompt = `Kamu adalah context analyzer. Analisis apakah pesan user adalah permintaan melanjutkan output sebelumnya ATAU query mandiri baru.

CONVERSATION HISTORY:
%s

USER MESSAGE: %s

RULES:
1. Output CONTINUE jika user meminta melanjutkan/sambung/terusin/next/more/lanjut/continue (bahasa apa pun, gaya apa pun, termasuk "terus gimana", "lanjut dong", "next", "more", "sambung", dll)
2. Output STANDALONE + tulis ulang query agar self-contained jika ini kebutuhan informasi baru atau pertanyaan berbeda
3. Jangan hardcode keyword — pahami MAKNA dari konteks percakapan
4. Jika history kosong atau user message tidak ada kaitan dengan history, output STANDALONE

EXAMPLES:

Example 1:
History:
assistant: "Dokumen ini membahas tentang machine learning. Ada beberapa konsep penting seperti supervised learning, unsupervised learning..."
user: "lanjutkan"
Output: {"type": "CONTINUE"}

Example 2:
History:
assistant: "Untuk membuat nasi goreng, pertama siapkan bahan-bahan: nasi putih, telur, bawang merah..."
user: "terus gimana"
Output: {"type": "CONTINUE"}

Example 3:
History:
assistant: "Python adalah bahasa pemrograman yang populer untuk data science..."
user: "apa itu JavaScript?"
Output: {"type": "STANDALONE", "rewritten_query": "Apa itu JavaScript?"}

Example 4:
History:
assistant: "Resep kue brownies membutuhkan coklat, telur, gula, dan tepung. Langkah pertama..."
user: "lanjut dong"
Output: {"type": "CONTINUE"}

Example 5:
History: (empty)
user: "jelaskan tentang AI"
Output: {"type": "STANDALONE", "rewritten_query": "Jelaskan tentang AI"}

Example 6:
History:
user: "siapa presiden Indonesia?"
assistant: "Presiden Indonesia saat ini adalah..."
user: "berapa umurnya?"
Output: {"type": "STANDALONE", "rewritten_query": "Berapa umur Presiden Indonesia saat ini?"}

Return ONLY valid JSON, no other text:
{"type": "CONTINUE"} 
OR 
{"type": "STANDALONE", "rewritten_query": "..."}`

func (c *llmContextualizer) Analyze(ctx context.Context, userText string, history []HistoryMessage) (*AnalysisResult, error) {
	start := time.Now()

	var historyStr string
	if len(history) == 0 {
		historyStr = "(empty - no previous conversation)"
	} else {
		var sb strings.Builder
		for _, h := range history {
			sb.WriteString(fmt.Sprintf("%s: %s\n", h.Role, h.Content))
		}
		historyStr = sb.String()
	}

	prompt := fmt.Sprintf(contextualizationPrompt, historyStr, userText)

	messages := []llm.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := c.ollama.ChatJSON(ctx, messages)
	if err != nil {
		c.logger.Warn().Err(err).Msg("[contextualizer] LLM call failed, defaulting to STANDALONE")
		return &AnalysisResult{
			Type:           TypeStandalone,
			RewrittenQuery: userText,
		}, nil
	}

	reply = strings.TrimSpace(reply)
	if reply == "" || reply == "{}" {
		c.logger.Warn().Msg("[contextualizer] empty response, defaulting to STANDALONE")
		return &AnalysisResult{
			Type:           TypeStandalone,
			RewrittenQuery: userText,
		}, nil
	}

	var result struct {
		Type           string `json:"type"`
		RewrittenQuery string `json:"rewritten_query,omitempty"`
	}

	if err := json.Unmarshal([]byte(reply), &result); err != nil {
		c.logger.Warn().Err(err).Str("reply", reply).Msg("[contextualizer] JSON parse failed, defaulting to STANDALONE")
		return &AnalysisResult{
			Type:           TypeStandalone,
			RewrittenQuery: userText,
		}, nil
	}

	analysisType := TypeStandalone
	if strings.ToUpper(result.Type) == "CONTINUE" {
		analysisType = TypeContinue
	}

	rewrittenQuery := result.RewrittenQuery
	if rewrittenQuery == "" {
		rewrittenQuery = userText
	}

	c.logger.Info().
		Str("type", string(analysisType)).
		Str("rewritten_query", rewrittenQuery).
		Dur("ms", time.Since(start)).
		Msg("[contextualizer] analysis done")

	return &AnalysisResult{
		Type:           analysisType,
		RewrittenQuery: rewrittenQuery,
	}, nil
}
