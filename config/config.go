package config

import (
	"os"
)

// LLMProvider specifies which LLM backend to use
type LLMProvider string

const (
	LLMProviderGemini LLMProvider = "gemini"
	LLMProviderLocal  LLMProvider = "local"
)

// Config holds the application configuration.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string
	DatabaseURL string
	// LLMProvider specifies which LLM to use: "gemini" or "local"
	LLMProvider LLMProvider
	// GoogleAPIKey is the API key for Gemini (required if LLMProvider is "gemini")
	GoogleAPIKey string
	// Model is the model name to use
	Model string
	// LocalLLMURL is the URL for local LLM server (e.g., "http://localhost:1234")
	LocalLLMURL string
	// MCPServerAddr is the address for the MCP server
	MCPServerAddr string
}

// New creates a new Config from environment variables.
func New() *Config {
	provider := LLMProvider(getEnvOrDefault("LLM_PROVIDER", "gemini"))

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		if provider == LLMProviderGemini {
			model = "gemini-2.0-flash"
		} else {
			model = "local-model"
		}
	}

	return &Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		LLMProvider:   provider,
		GoogleAPIKey:  os.Getenv("GOOGLE_API_KEY"),
		Model:         model,
		LocalLLMURL:   getEnvOrDefault("LOCAL_LLM_URL", "http://localhost:1234"),
		MCPServerAddr: getEnvOrDefault("MCP_SERVER_ADDR", "localhost:9000"),
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return ErrMissingDatabaseURL
	}
	if c.LLMProvider == LLMProviderGemini && c.GoogleAPIKey == "" {
		return ErrMissingAPIKey
	}
	if c.LLMProvider == LLMProviderLocal && c.LocalLLMURL == "" {
		return ErrMissingLocalLLMURL
	}
	return nil
}

// IsLocalLLM returns true if using a local LLM
func (c *Config) IsLocalLLM() bool {
	return c.LLMProvider == LLMProviderLocal
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// Error definitions
type ConfigError string

func (e ConfigError) Error() string { return string(e) }

const (
	ErrMissingDatabaseURL ConfigError = "DATABASE_URL environment variable is required"
	ErrMissingAPIKey      ConfigError = "GOOGLE_API_KEY environment variable is required when using Gemini"
	ErrMissingLocalLLMURL ConfigError = "LOCAL_LLM_URL environment variable is required when using local LLM"
)
