// Package reasoning implements post-generation verification and consistency checking.
// Before sending a response to the user, the reasoning layer validates it
// against conversation history, user corrections, and known facts (Principles 9, 11, 12, 17).
package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/rs/zerolog"
)

// ─── Types ─────────────────────────────────────────────────────────

// VerificationResult holds the outcome of response verification.
type VerificationResult struct {
	IsValid           bool     `json:"is_valid"`
	ConfidenceScore   float32  `json:"confidence_score"` // 0.0 to 1.0
	Issues            []string `json:"issues"`
	Suggestions       []string `json:"suggestions"`
	FactsToCheck      []string `json:"facts_to_check"`
	ContradictsPast   bool     `json:"contradicts_past"`
	ContradictsFacts  bool     `json:"contradicts_facts"`
	HallucinationRisk string   `json:"hallucination_risk"` // "low", "medium", "high"
}

// ConsistencyCheck validates response against history and memory.
type ConsistencyCheck struct {
	Consistent         bool     `json:"consistent"`
	Contradictions     []string `json:"contradictions"`
	PreviousStatements []string `json:"previous_statements"`
}

// ─── Interface ─────────────────────────────────────────────────────

// Verifier validates LLM responses before sending to user.
type Verifier interface {
	// Verify checks response for correctness, consistency, and hallucinations.
	Verify(ctx context.Context, params VerifyParams) (*VerificationResult, error)
	
	// CheckConsistency validates response against conversation history.
	CheckConsistency(ctx context.Context, params ConsistencyParams) (*ConsistencyCheck, error)
}

// VerifyParams holds inputs for verification.
type VerifyParams struct {
	UserID       string
	SessionID    string
	UserQuestion string
	Response     string
	State        *domain.StateData
	RAGContext   string
	RAGSources   []string
}

// ConsistencyParams holds inputs for consistency checking.
type ConsistencyParams struct {
	UserID    string
	SessionID string
	Response  string
	HistoryLimit int
}

// ─── Implementation ────────────────────────────────────────────────

type verifier struct {
	llm    *ollama.OllamaClient
	repo   repository.Repository
	memory factmemory.MemoryManager
	logger *zerolog.Logger
}

// NewVerifier creates a Verifier.
func NewVerifier(
	llm *ollama.OllamaClient,
	repo repository.Repository,
	memory factmemory.MemoryManager,
	logger *zerolog.Logger,
) Verifier {
	return &verifier{
		llm:    llm,
		repo:   repo,
		memory: memory,
		logger: logger,
	}
}

func (v *verifier) Verify(ctx context.Context, params VerifyParams) (*VerificationResult, error) {
	start := time.Now()

	// Fast path: if response is very short or a simple acknowledgment, skip verification
	if len(params.Response) < 50 || v.isSimpleAck(params.Response) {
		v.logger.Debug().Msg("[verifier] skipping verification for simple response")
		return &VerificationResult{
			IsValid:           true,
			ConfidenceScore:   1.0,
			HallucinationRisk: "low",
		}, nil
	}

	// Check for recent corrections
	hasRecentCorrection := false
	if params.State != nil && params.State.LastCorrectionAt != "" {
		corrTime, _ := time.Parse(time.RFC3339, params.State.LastCorrectionAt)
		if time.Since(corrTime) < 5*time.Minute {
			hasRecentCorrection = true
		}
	}

	// Build verification prompt
	prompt := v.buildVerificationPrompt(params, hasRecentCorrection)

	messages := []ollama.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := v.llm.ChatJSON(ctx, messages)
	if err != nil {
		v.logger.Warn().Err(err).Msg("[verifier] LLM verification failed, using fallback")
		return v.fallbackVerification(params), nil
	}

	var result VerificationResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(reply)), &result); err != nil {
		v.logger.Warn().Err(err).Msg("[verifier] parse verification failed")
		return v.fallbackVerification(params), nil
	}

	v.logger.Info().
		Bool("valid", result.IsValid).
		Float32("confidence", result.ConfidenceScore).
		Str("hallucination_risk", result.HallucinationRisk).
		Int("issues", len(result.Issues)).
		Dur("ms", time.Since(start)).
		Msg("[verifier] verification complete")

	return &result, nil
}

func (v *verifier) CheckConsistency(ctx context.Context, params ConsistencyParams) (*ConsistencyCheck, error) {
	start := time.Now()

	// Fetch recent history
	history, err := v.repo.ListMessagesByUserSession(ctx, params.UserID, params.SessionID, params.HistoryLimit)
	if err != nil || len(history) == 0 {
		v.logger.Debug().Msg("[verifier] no history for consistency check")
		return &ConsistencyCheck{Consistent: true}, nil
	}

	// Build history context
	var historyText strings.Builder
	for _, msg := range history {
		historyText.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(msg.Role), msg.Content))
	}

	prompt := fmt.Sprintf(`Analyze if this new response is CONSISTENT with the conversation history.

CONVERSATION HISTORY:
%s

NEW RESPONSE TO CHECK:
%s

Return ONLY valid JSON:
{
  "consistent": true/false,
  "contradictions": ["list of any contradictions found"],
  "previous_statements": ["relevant previous statements that contradict"]
}`, historyText.String(), params.Response)

	messages := []ollama.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := v.llm.ChatJSON(ctx, messages)
	if err != nil {
		v.logger.Warn().Err(err).Msg("[verifier] consistency check failed")
		return &ConsistencyCheck{Consistent: true}, nil // assume consistent on error
	}

	var result ConsistencyCheck
	if err := json.Unmarshal([]byte(strings.TrimSpace(reply)), &result); err != nil {
		v.logger.Warn().Err(err).Msg("[verifier] parse consistency failed")
		return &ConsistencyCheck{Consistent: true}, nil
	}

	v.logger.Info().
		Bool("consistent", result.Consistent).
		Int("contradictions", len(result.Contradictions)).
		Dur("ms", time.Since(start)).
		Msg("[verifier] consistency check complete")

	return &result, nil
}

// ─── Helpers ───────────────────────────────────────────────────────

func (v *verifier) buildVerificationPrompt(params VerifyParams, hasRecentCorrection bool) string {
	var sb strings.Builder

	sb.WriteString("You are a verification agent. Analyze this response for correctness and quality.\n\n")

	sb.WriteString(fmt.Sprintf("USER QUESTION: %s\n\n", params.UserQuestion))
	sb.WriteString(fmt.Sprintf("RESPONSE TO VERIFY: %s\n\n", params.Response))

	if params.RAGContext != "" {
		sb.WriteString("AVAILABLE CONTEXT (from documents):\n")
		sb.WriteString(params.RAGContext)
		sb.WriteString("\n\n")
	}

	if hasRecentCorrection {
		sb.WriteString("⚠️ USER RECENTLY CORRECTED A PREVIOUS RESPONSE. Be extra careful about consistency.\n\n")
	}

	sb.WriteString(`VERIFICATION CHECKLIST:
1. Does the response answer the user's question?
2. Is the response grounded in the provided context (if any)?
3. Are there any unsupported claims or hallucinations?
4. Is the response consistent with itself?
5. Does it contradict any known facts about the user?

Return ONLY valid JSON:
{
  "is_valid": true/false,
  "confidence_score": 0.0-1.0,
  "issues": ["list of problems found"],
  "suggestions": ["how to improve the response"],
  "facts_to_check": ["claims that need verification"],
  "contradicts_past": true/false,
  "contradicts_facts": true/false,
  "hallucination_risk": "low/medium/high"
}`)

	return sb.String()
}

func (v *verifier) isSimpleAck(response string) bool {
	lower := strings.ToLower(strings.TrimSpace(response))
	simple := []string{"ok", "oke", "baik", "siap", "ya", "yes", "got it", "understood", "noted"}
	for _, s := range simple {
		if lower == s || lower == s+"." || lower == s+"!" {
			return true
		}
	}
	// Check for emoji-only or very short responses
	if len(response) <= 10 && (strings.Contains(response, "✅") || strings.Contains(response, "👍")) {
		return true
	}
	return false
}

func (v *verifier) fallbackVerification(params VerifyParams) *VerificationResult {
	// Simple heuristic-based verification
	issues := []string{}
	hallucinationRisk := "low"
	
	// Check if response claims certainty without context
	if params.RAGContext == "" && v.containsCertainLanguage(params.Response) {
		issues = append(issues, "Response claims certainty without supporting context")
		hallucinationRisk = "medium"
	}

	// Check length mismatch
	if len(params.UserQuestion) < 50 && len(params.Response) > 500 {
		issues = append(issues, "Response may be overly verbose for simple question")
	}

	isValid := len(issues) == 0
	confidence := float32(0.7)
	if !isValid {
		confidence = 0.5
	}

	return &VerificationResult{
		IsValid:           isValid,
		ConfidenceScore:   confidence,
		Issues:            issues,
		HallucinationRisk: hallucinationRisk,
	}
}

func (v *verifier) containsCertainLanguage(text string) bool {
	certainWords := []string{
		"definitely", "certainly", "absolutely", "pasti", "tentu", "jelas",
		"without doubt", "tanpa ragu", "100%", "always", "selalu",
	}
	lower := strings.ToLower(text)
	for _, word := range certainWords {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}
