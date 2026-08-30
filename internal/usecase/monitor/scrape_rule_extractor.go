package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/pkg/llm"
)

type ScrapeRuleExtractor struct {
	llmClient llm.Client
	ruleRepo  domain.ExtractionRuleRepository
	client    *http.Client
}

func NewScrapeRuleExtractor(llmClient llm.Client, ruleRepo domain.ExtractionRuleRepository) *ScrapeRuleExtractor {
	return &ScrapeRuleExtractor{
		llmClient: llmClient,
		ruleRepo:  ruleRepo,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

type extractionRuleResponse struct {
	Selector string `json:"selector"`
	RuleType string `json:"rule_type"`
}

func (e *ScrapeRuleExtractor) DeriveExtractionRule(ctx context.Context, taskID int64, url string, description string) (domain.ExtractionRule, error) {
	html, err := e.fetchHTML(ctx, url)
	if err != nil {
		return domain.ExtractionRule{}, fmt.Errorf("fetch html: %w", err)
	}

	trimmed := trimHTML(html)

	rule, err := e.askLLM(ctx, trimmed, description)
	if err != nil {
		return domain.ExtractionRule{}, fmt.Errorf("llm derive: %w", err)
	}

	if err := e.validate(html, rule); err != nil {
		rule, err = e.askLLMRetry(ctx, trimmed, description, err.Error())
		if err != nil {
			return domain.ExtractionRule{}, fmt.Errorf("llm retry: %w", err)
		}
		if err := e.validate(html, rule); err != nil {
			return domain.ExtractionRule{}, fmt.Errorf("validation failed after retry: %w", err)
		}
	}

	result := domain.ExtractionRule{
		MonitorTaskID: taskID,
		Selector:      rule.Selector,
		RuleType:      rule.RuleType,
		DerivedAt:     time.Now(),
	}
	
	if taskID > 0 {
		if err := e.ruleRepo.Save(ctx, result); err != nil {
			return domain.ExtractionRule{}, fmt.Errorf("save rule: %w", err)
		}
	}
	
	return result, nil
}

func (e *ScrapeRuleExtractor) fetchHTML(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (e *ScrapeRuleExtractor) askLLM(ctx context.Context, html string, description string) (extractionRuleResponse, error) {
	prompt := fmt.Sprintf(`You are given HTML content and a description of what data to extract.
Return a JSON object with exactly two fields:
- "selector": a CSS selector or regex pattern to extract the value
- "rule_type": either "css_selector" or "regex"

Description of data to extract: %s

HTML content (trimmed):
%s

Respond ONLY with the JSON object, no other text.`, description, html)

	resp, err := e.llmClient.ChatJSON(ctx, []llm.ChatMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return extractionRuleResponse{}, err
	}

	var result extractionRuleResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return extractionRuleResponse{}, fmt.Errorf("parse llm response: %w", err)
	}
	if result.Selector == "" || result.RuleType == "" {
		return extractionRuleResponse{}, fmt.Errorf("empty selector or rule_type from LLM")
	}
	return result, nil
}

func (e *ScrapeRuleExtractor) askLLMRetry(ctx context.Context, html string, description string, prevError string) (extractionRuleResponse, error) {
	prompt := fmt.Sprintf(`Previous extraction rule failed with error: %s

Try a different approach. You are given HTML content and a description of what data to extract.
Return a JSON object with exactly two fields:
- "selector": a CSS selector or regex pattern to extract the value
- "rule_type": either "css_selector" or "regex"

Description: %s

HTML content (trimmed):
%s

Respond ONLY with the JSON object, no other text.`, prevError, description, html)

	resp, err := e.llmClient.ChatJSON(ctx, []llm.ChatMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return extractionRuleResponse{}, err
	}

	var result extractionRuleResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return extractionRuleResponse{}, fmt.Errorf("parse llm retry response: %w", err)
	}
	return result, nil
}

func (e *ScrapeRuleExtractor) validate(html string, rule extractionRuleResponse) error {
	switch rule.RuleType {
	case "css_selector":
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return err
		}
		if doc.Find(rule.Selector).Length() == 0 {
			return fmt.Errorf("css selector %q matched 0 elements", rule.Selector)
		}
	case "regex":
		if _, err := regexp.Compile(rule.Selector); err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	default:
		return fmt.Errorf("unknown rule_type: %s", rule.RuleType)
	}
	return nil
}

func trimHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		if len(html) > 8000 {
			return html[:8000]
		}
		return html
	}
	doc.Find("script, style, noscript, iframe, svg, link, meta").Remove()
	result, _ := doc.Html()
	if len(result) > 8000 {
		return result[:8000]
	}
	return result
}
