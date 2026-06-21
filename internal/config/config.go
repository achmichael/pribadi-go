package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all application configuration
type Config struct {
	AppPort          string `envconfig:"APP_PORT" required:"true"`
	LogLevel         string `envconfig:"LOG_LEVEL" default:"info"`
	SQLitePath       string `envconfig:"SQLITE_PATH" required:"true"`
	VectorDBPath     string `envconfig:"VECTOR_DB_PATH" required:"true"`
	OllamaBaseURL    string `envconfig:"OLLAMA_BASE_URL" required:"true"`
	OllamaModel      string `envconfig:"OLLAMA_MODEL" required:"true"`
	WhisperModelPath string `envconfig:"WHISPER_MODEL_PATH" required:"true"`
	WhisperBinPath   string `envconfig:"WHISPER_BIN_PATH" required:"true"`
	LocalDiskPath    string `envconfig:"LOCAL_DISK_PATH" required:"true"`
	WebhookSecret    string `envconfig:"WEBHOOK_SECRET" default:""`
	NotifyJIDs       string `envconfig:"NOTIFY_JIDS" required:"true"`
	FlorenceURL      string `envconfig:"FLORENCE_URL" default:"http://localhost:8100"`
}

// Load reads configuration from .env file and environment variables
func Load() (*Config, error) {
	// Load .env file if exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process config: %w", err)
	}

	return &cfg, nil
}
