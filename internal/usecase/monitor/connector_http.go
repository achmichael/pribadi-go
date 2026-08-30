package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/pkg/crypto"
	"github.com/tidwall/gjson"
)

type HTTPConnector struct {
	client        *http.Client
	encryptionKey []byte
}

func NewHTTPConnector(encryptionKey string) *HTTPConnector {
	return &HTTPConnector{
		client:        &http.Client{Timeout: 30 * time.Second},
		encryptionKey: []byte(encryptionKey),
	}
}

func (c *HTTPConnector) Fetch(ctx context.Context, task domain.MonitorTask) (*float64, string, string, error) {
	var cfg domain.HTTPSourceConfig
	if err := json.Unmarshal([]byte(task.SourceConfig), &cfg); err != nil {
		return nil, "", "", fmt.Errorf("parse source_config: %w", err)
	}

	method := strings.ToUpper(cfg.Method)
	if method == "" {
		method = "GET"
	}

	var bodyReader io.Reader
	if cfg.Body != "" {
		bodyReader = strings.NewReader(cfg.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, bodyReader)
	if err != nil {
		return nil, "", "", fmt.Errorf("create request: %w", err)
	}

	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}

	if err := c.applyAuth(req, cfg); err != nil {
		return nil, "", "", fmt.Errorf("apply auth: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", fmt.Errorf("read body: %w", err)
	}
	rawSnapshot := string(body)

	if resp.StatusCode >= 400 {
		return nil, "", rawSnapshot, fmt.Errorf("http status %d", resp.StatusCode)
	}

	if cfg.ExtractPath == "" {
		return nil, rawSnapshot, rawSnapshot, nil
	}

	result := gjson.Get(rawSnapshot, cfg.ExtractPath)
	if !result.Exists() {
		return nil, "", rawSnapshot, fmt.Errorf("extract_path %q not found in response", cfg.ExtractPath)
	}

	text := result.String()

	if cfg.ExtractAsText {
		return nil, text, rawSnapshot, nil
	}

	val, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, text, rawSnapshot, nil
	}
	return &val, text, rawSnapshot, nil
}

func (c *HTTPConnector) applyAuth(req *http.Request, cfg domain.HTTPSourceConfig) error {
	if cfg.AuthType == "" || cfg.AuthType == "none" || cfg.AuthValue == "" {
		return nil
	}

	authValue, err := c.decryptAuth(cfg.AuthValue)
	if err != nil {
		return err
	}

	switch cfg.AuthType {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+authValue)
	case "api_key_header":
		parts := strings.SplitN(authValue, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(parts[0], parts[1])
		}
	case "api_key_query":
		parts := strings.SplitN(authValue, ":", 2)
		if len(parts) == 2 {
			q := req.URL.Query()
			q.Set(parts[0], parts[1])
			req.URL.RawQuery = q.Encode()
		}
	}
	return nil
}

func (c *HTTPConnector) decryptAuth(encrypted string) (string, error) {
	if len(c.encryptionKey) == 0 {
		return encrypted, nil
	}
	plain, err := crypto.Decrypt(encrypted, c.encryptionKey)
	if err != nil {
		return encrypted, nil
	}
	return string(plain), nil
}
