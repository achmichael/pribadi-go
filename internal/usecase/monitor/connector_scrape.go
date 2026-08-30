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
)

const ruleFailThreshold = 3

type ScrapeConnector struct {
	client    *http.Client
	ruleRepo  domain.ExtractionRuleRepository
	extractor *ScrapeRuleExtractor
}

func NewScrapeConnector(ruleRepo domain.ExtractionRuleRepository, extractor *ScrapeRuleExtractor) *ScrapeConnector {
	return &ScrapeConnector{
		client:    &http.Client{Timeout: 30 * time.Second},
		ruleRepo:  ruleRepo,
		extractor: extractor,
	}
}

func (c *ScrapeConnector) Fetch(ctx context.Context, task domain.MonitorTask) (*float64, string, string, error) {
	var cfg domain.ScrapeSourceConfig
	if err := json.Unmarshal([]byte(task.SourceConfig), &cfg); err != nil {
		return nil, "", "", fmt.Errorf("parse source_config: %w", err)
	}

	rule, err := c.ruleRepo.GetByTaskID(ctx, task.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("get extraction rule: %w", err)
	}

	if rule == nil || rule.FailCount >= ruleFailThreshold {
		newRule, err := c.extractor.DeriveExtractionRule(ctx, task.ID, cfg.URL, cfg.Description)
		if err != nil {
			return nil, "", "", fmt.Errorf("derive rule: %w", err)
		}
		rule = &newRule
	}

	html, err := c.fetchHTML(ctx, cfg.URL)
	if err != nil {
		return nil, "", "", fmt.Errorf("fetch html: %w", err)
	}

	text, err := c.applyRule(html, *rule)
	if err != nil {
		_ = c.ruleRepo.IncrementFailCount(ctx, task.ID)
		return nil, "", html, fmt.Errorf("apply rule: %w", err)
	}

	text = strings.TrimSpace(text)
	return nil, text, html, nil
}

func (c *ScrapeConnector) fetchHTML(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.client.Do(req)
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

func (c *ScrapeConnector) applyRule(html string, rule domain.ExtractionRule) (string, error) {
	switch rule.RuleType {
	case "css_selector":
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return "", err
		}
		sel := doc.Find(rule.Selector)
		if sel.Length() == 0 {
			return "", fmt.Errorf("selector %q matched 0 elements", rule.Selector)
		}
		return sel.First().Text(), nil
	case "regex":
		re, err := regexp.Compile(rule.Selector)
		if err != nil {
			return "", fmt.Errorf("compile regex: %w", err)
		}
		match := re.FindStringSubmatch(html)
		if len(match) < 2 {
			if len(match) == 1 {
				return match[0], nil
			}
			return "", fmt.Errorf("regex %q no match", rule.Selector)
		}
		return match[1], nil
	default:
		return "", fmt.Errorf("unknown rule_type: %s", rule.RuleType)
	}
}
