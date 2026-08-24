package budget

import "strings"

const safetyMargin = 0.85

func EstimateTokens(text string) int {
	words := len(strings.Fields(text))
	return int(float64(words) * 1.3)
}

type Calculator struct {
	NumCtx     int
	NumPredict int
}

func NewCalculator(numCtx, numPredict int) *Calculator {
	return &Calculator{NumCtx: numCtx, NumPredict: numPredict}
}

func (c *Calculator) RemainingBudget(systemPrompt string, historyTexts []string) int {
	used := EstimateTokens(systemPrompt) + c.NumPredict
	for _, h := range historyTexts {
		used += EstimateTokens(h)
	}
	maxUsable := int(float64(c.NumCtx) * safetyMargin)
	remaining := maxUsable - used
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (c *Calculator) FitsInContext(fileTokens int, systemPrompt string, historyTexts []string) bool {
	return fileTokens <= c.RemainingBudget(systemPrompt, historyTexts)
}
