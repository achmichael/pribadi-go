package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/rs/zerolog"
)

// ─── Types ─────────────────────────────────────────────────────────

// FactCheck represents validation of a claim.
type FactCheck struct {
	Claim          string  `json:"claim"`
	IsSupported    bool    `json:"is_supported"`
	Evidence       string  `json:"evidence"`
	Confidence     float32 `json:"confidence"`
	NeedsExternal  bool    `json:"needs_external"` // needs web search or external verification
}

// FactCheckResult holds results of checking all claims in a response.
type FactCheckResult struct {
	TotalClaims      int         `json:"total_claims"`
	SupportedClaims  int         `json:"supported_claims"`
	UnsupportedClaims int        `json:"unsupported_claims"`
	Checks           []FactCheck `json:"checks"`
	OverallValid     bool        `json:"overall_valid"`
}

// ─── Interface ─────────────────────────────────────────────────────

// FactChecker validates factual claims in responses.
type FactChecker interface {
	// Check validates claims against known context and memory.
	Check(ctx context.Context, params FactCheckParams) (*FactCheckResult, error)
}

// FactCheckParams holds inputs for fact checking.
type FactCheckParams struct {
	UserID     string
	Response   string
	RAGContext string
	UserQuery  string
}

// ─── Implementation ────────────────────────────────────────────────

type factChecker struct {
	llm    *ollama.OllamaClient
	memory factmemory.MemoryManager
	logger *zerolog.Logger
}

// NewFactChecker creates a FactChecker.
func NewFactChecker(
	llm *ollama.OllamaClient,
	memory factmemory.MemoryManager,
	logger *zerolog.Logger,
) FactChecker {
	return &factChecker{
		llm:    llm,
		memory: memory,
		logger: logger,
	}
}

func (c *factChecker) Check(ctx context.Context, params FactCheckParams) (*FactCheckResult, error) {
	start := time.Now()

	// Extract claims from response
	claims, err := c.extractClaims(ctx, params.Response)
	if err != nil || len(claims) == 0 {
		c.logger.Debug().Msg("[factchecker] no claims to check")
		return &FactCheckResult{OverallValid: true}, nil
	}

	// Fetch user memory for cross-checking
	memoryContext := ""
	if params.UserID != "" {
		memoryContext, _ = c.memory.PrefetchRelevant(ctx, params.UserID, params.UserQuery)
	}

	// Check each claim
	checks := make([]FactCheck, 0, len(claims))
	supported := 0
	unsupported := 0

	for _, claim := range claims {
		check, err := c.checkClaim(ctx, claim, params.RAGContext, memoryContext)
		if err != nil {
			c.logger.Warn().Err(err).Str("claim", claim).Msg("[factchecker] check failed")
			continue
		}
		checks = append(checks, *check)
		if check.IsSupported {
			supported++
		} else {
			unsupported++
		}
	}

	// Overall valid if most claims are supported
	overallValid := true
	if len(checks) > 0 {
		supportRatio := float32(supported) / float32(len(checks))
		overallValid = supportRatio >= 0.7
	}

	result := &FactCheckResult{
		TotalClaims:       len(checks),
		SupportedClaims:   supported,
		UnsupportedClaims: unsupported,
		Checks:            checks,
		OverallValid:      overallValid,
	}

	c.logger.Info().
		Int("total", result.TotalClaims).
		Int("supported", supported).
		Int("unsupported", unsupported).
		Bool("valid", overallValid).
		Dur("ms", time.Since(start)).
		Msg("[factchecker] check complete")

	return result, nil
}

// extractClaims identifies factual claims in the response.
func (c *factChecker) extractClaims(ctx context.Context, response string) ([]string, error) {
	prompt := fmt.Sprintf(`Extract factual claims from this response. Ignore opinions and subjective statements.

RESPONSE: %s

Return ONLY a JSON array of strings, each containing one factual claim:
["claim 1", "claim 2", ...]

If no factual claims, return empty array: []`, response)

	messages := []ollama.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := c.llm.ChatJSON(ctx, messages)
	if err != nil {
		return nil, err
	}

	var claims []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(reply)), &claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// checkClaim validates a single claim against available context.
func (c *factChecker) checkClaim(ctx context.Context, claim, ragContext, memoryContext string) (*FactCheck, error) {
	prompt := fmt.Sprintf(`Verify if this claim is supported by the available context.

CLAIM TO CHECK: %s

AVAILABLE CONTEXT (documents):
%s

USER MEMORY:
%s

Return ONLY valid JSON:
{
  "claim": "%s",
  "is_supported": true/false,
  "evidence": "quote from context that supports/contradicts the claim",
  "confidence": 0.0-1.0,
  "needs_external": true/false
}`, claim, ragContext, memoryContext, claim)

	messages := []ollama.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := c.llm.ChatJSON(ctx, messages)
	if err != nil {
		return nil, err
	}

	var result FactCheck
	if err := json.Unmarshal([]byte(strings.TrimSpace(reply)), &result); err != nil {
		return nil, err
	}

	return &result, nil
}
