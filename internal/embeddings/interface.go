package embeddings

import (
	"context"
)

// EmbeddingClient is the interface for embedding generation
type EmbeddingClient interface {
	// Embed generates an embedding for a single text
	Embed(ctx context.Context, text string) (*EmbeddingResult, error)

	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) (*BatchResult, error)

	// GetStats returns client statistics
	GetStats() Stats

	// CacheSize returns the number of cached entries
	CacheSize() int

	// ClearCache clears the embedding cache
	ClearCache()
}

// Ensure both clients implement the interface
var _ EmbeddingClient = (*Client)(nil)
var _ EmbeddingClient = (*OllamaClient)(nil)

// ProviderType represents the embedding provider
type ProviderType string

const (
	ProviderOpenAI ProviderType = "openai"
	ProviderOllama ProviderType = "ollama"
)

// UnifiedConfig holds configuration for any embedding provider
type UnifiedConfig struct {
	Provider     ProviderType  `json:"provider"`      // "openai" or "ollama"
	APIKey       string        `json:"api_key"`       // For OpenAI
	BaseURL      string        `json:"base_url"`      // Custom endpoint
	Model        string        `json:"model"`         // Model name
	RateLimit    int           `json:"rate_limit"`    // Requests per minute
	BatchSize    int           `json:"batch_size"`    // Texts per batch
}

// NewEmbeddingClient creates an embedding client based on configuration
func NewEmbeddingClient(config *UnifiedConfig) (EmbeddingClient, error) {
	if config == nil {
		// Default to Ollama for offline use
		return NewOllamaClient(DefaultOllamaConfig())
	}

	switch config.Provider {
	case ProviderOllama:
		ollamaConfig := DefaultOllamaConfig()
		if config.BaseURL != "" {
			ollamaConfig.BaseURL = config.BaseURL
		}
		if config.Model != "" {
			ollamaConfig.Model = config.Model
		}
		if config.RateLimit > 0 {
			ollamaConfig.RateLimit = config.RateLimit
		}
		if config.BatchSize > 0 {
			ollamaConfig.BatchSize = config.BatchSize
		}
		return NewOllamaClient(ollamaConfig)

	case ProviderOpenAI:
		fallthrough
	default:
		openaiConfig := DefaultClientConfig(config.APIKey)
		if config.BaseURL != "" {
			openaiConfig.BaseURL = config.BaseURL
		}
		if config.Model != "" {
			openaiConfig.Model = config.Model
		}
		if config.RateLimit > 0 {
			openaiConfig.RateLimit = config.RateLimit
		}
		if config.BatchSize > 0 {
			openaiConfig.BatchSize = config.BatchSize
		}
		return NewClient(openaiConfig)
	}
}

// DetectProvider tries to detect the provider from configuration
func DetectProvider(baseURL, apiKey string) ProviderType {
	// If base URL contains ollama, use Ollama
	if baseURL != "" {
		if contains(baseURL, "localhost:11434") || contains(baseURL, "ollama") {
			return ProviderOllama
		}
	}

	// If no API key, default to Ollama (offline)
	if apiKey == "" || apiKey == "ollama" {
		return ProviderOllama
	}

	return ProviderOpenAI
}
