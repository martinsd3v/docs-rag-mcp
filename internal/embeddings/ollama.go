package embeddings

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// OllamaDefaultURL is the default Ollama API endpoint
	OllamaDefaultURL = "http://localhost:11434"

	// OllamaDefaultModel is the default embedding model for Ollama
	OllamaDefaultModel = "nomic-embed-text"

	// NomicEmbedDimension is the dimension of nomic-embed-text vectors
	NomicEmbedDimension = 768

	// MxbaiEmbedDimension is the dimension of mxbai-embed-large vectors
	MxbaiEmbedDimension = 1024
)

// OllamaConfig holds configuration for the Ollama client
type OllamaConfig struct {
	BaseURL      string        `json:"base_url"`
	Model        string        `json:"model"`
	RateLimit    int           `json:"rate_limit"`    // requests per minute
	BatchSize    int           `json:"batch_size"`    // texts per batch
	MaxRetries   int           `json:"max_retries"`   // max retry attempts
	Timeout      time.Duration `json:"timeout"`       // request timeout
	RetryBackoff []time.Duration `json:"retry_backoff"` // backoff durations
}

// DefaultOllamaConfig returns default Ollama configuration
func DefaultOllamaConfig() *OllamaConfig {
	return &OllamaConfig{
		BaseURL:    OllamaDefaultURL,
		Model:      OllamaDefaultModel,
		RateLimit:  600, // Ollama is local, can be faster
		BatchSize:  50,  // Process 50 at a time
		MaxRetries: 3,
		Timeout:    60 * time.Second,
		RetryBackoff: []time.Duration{
			500 * time.Millisecond,
			1 * time.Second,
			2 * time.Second,
		},
	}
}

// OllamaEmbeddingRequest is the request format for Ollama API
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbeddingResponse is the response format from Ollama API
type OllamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// OllamaClient wraps the Ollama API for embedding generation
type OllamaClient struct {
	config     *OllamaConfig
	httpClient *http.Client
	limiter    *rate.Limiter
	cache      map[string]*CacheEntry
	cacheMutex sync.RWMutex
	stats      *Stats
	dimension  int
}

// NewOllamaClient creates a new Ollama embedding client
func NewOllamaClient(config *OllamaConfig) (*OllamaClient, error) {
	if config == nil {
		config = DefaultOllamaConfig()
	}

	// Set defaults if not specified
	if config.BaseURL == "" {
		config.BaseURL = OllamaDefaultURL
	}
	if config.Model == "" {
		config.Model = OllamaDefaultModel
	}
	if config.RateLimit == 0 {
		config.RateLimit = 600
	}
	if config.BatchSize == 0 {
		config.BatchSize = 50
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if len(config.RetryBackoff) == 0 {
		config.RetryBackoff = []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second}
	}

	// Determine embedding dimension based on model
	dimension := NomicEmbedDimension
	if config.Model == "mxbai-embed-large" {
		dimension = MxbaiEmbedDimension
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Create rate limiter (requests per second)
	rps := float64(config.RateLimit) / 60.0
	limiter := rate.NewLimiter(rate.Limit(rps), config.BatchSize)

	return &OllamaClient{
		config:     config,
		httpClient: httpClient,
		limiter:    limiter,
		cache:      make(map[string]*CacheEntry),
		stats:      &Stats{},
		dimension:  dimension,
	}, nil
}

// GetDimension returns the embedding dimension for the configured model
func (c *OllamaClient) GetDimension() int {
	return c.dimension
}

// Embed generates an embedding for a single text
func (c *OllamaClient) Embed(ctx context.Context, text string) (*EmbeddingResult, error) {
	results, err := c.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(results.Results) == 0 {
		return nil, fmt.Errorf("no results returned")
	}
	return &results.Results[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (c *OllamaClient) EmbedBatch(ctx context.Context, texts []string) (*BatchResult, error) {
	start := time.Now()
	result := &BatchResult{
		Results: make([]EmbeddingResult, len(texts)),
	}

	// Check cache first
	textsToEmbed := make([]string, 0)
	textIndices := make([]int, 0)

	for i, text := range texts {
		// Skip empty strings
		if len(text) == 0 || text == "" {
			result.Results[i] = EmbeddingResult{
				Text:       text,
				Embedding:  make([]float32, c.dimension),
				TokenCount: 0,
				Cached:     false,
			}
			continue
		}

		hash := c.hashText(text)

		c.cacheMutex.RLock()
		entry, found := c.cache[hash]
		c.cacheMutex.RUnlock()

		if found {
			result.Results[i] = EmbeddingResult{
				Text:       text,
				Embedding:  entry.Embedding,
				TokenCount: entry.TokenCount,
				Cached:     true,
			}
			result.CacheHits++

			// Update cache stats
			c.cacheMutex.Lock()
			entry.LastUsed = time.Now()
			entry.UseCount++
			c.cacheMutex.Unlock()

			c.stats.mutex.Lock()
			c.stats.CacheHits++
			c.stats.mutex.Unlock()
		} else {
			textsToEmbed = append(textsToEmbed, text)
			textIndices = append(textIndices, i)
			result.CacheMisses++

			c.stats.mutex.Lock()
			c.stats.CacheMisses++
			c.stats.mutex.Unlock()
		}
	}

	// Generate embeddings for uncached texts
	if len(textsToEmbed) > 0 {
		embeddings, err := c.generateEmbeddings(ctx, textsToEmbed)
		if err != nil {
			return nil, err
		}

		// Store results and update cache
		for i, embedding := range embeddings {
			idx := textIndices[i]
			text := textsToEmbed[i]

			// Estimate tokens (rough: ~4 chars per token)
			tokenCount := len(text) / 4

			result.Results[idx] = EmbeddingResult{
				Text:       text,
				Embedding:  embedding,
				TokenCount: tokenCount,
				Cached:     false,
			}
			result.TotalTokens += tokenCount

			// Add to cache
			hash := c.hashText(text)
			c.cacheMutex.Lock()
			c.cache[hash] = &CacheEntry{
				Embedding:  embedding,
				TokenCount: tokenCount,
				Model:      c.config.Model,
				CreatedAt:  time.Now(),
				LastUsed:   time.Now(),
				UseCount:   1,
			}
			c.cacheMutex.Unlock()
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// generateEmbeddings calls the Ollama API for each text
func (c *OllamaClient) generateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		// Wait for rate limiter
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter error: %w", err)
		}

		embedding, err := c.embedWithRetry(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embedding %d failed: %w", i, err)
		}

		embeddings[i] = embedding

		c.stats.mutex.Lock()
		c.stats.TotalRequests++
		c.stats.mutex.Unlock()
	}

	return embeddings, nil
}

// embedWithRetry calls Ollama API with retry logic
func (c *OllamaClient) embedWithRetry(ctx context.Context, text string) ([]float32, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		embedding, err := c.callOllama(ctx, text)
		if err == nil {
			return embedding, nil
		}

		lastErr = err

		// Wait before retry
		if attempt < c.config.MaxRetries {
			backoff := c.config.RetryBackoff[min(attempt, len(c.config.RetryBackoff)-1)]
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}

			c.stats.mutex.Lock()
			c.stats.RetryCount++
			c.stats.mutex.Unlock()
		}
	}

	c.stats.mutex.Lock()
	c.stats.Errors++
	c.stats.mutex.Unlock()

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// callOllama makes the actual API call
func (c *OllamaClient) callOllama(ctx context.Context, text string) ([]float32, error) {
	// Build request
	reqBody := OllamaEmbeddingRequest{
		Model:  c.config.Model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/embeddings", c.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResp OllamaEmbeddingResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(ollamaResp.Embedding))
	for i, v := range ollamaResp.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// hashText creates a SHA256 hash of the text for caching
func (c *OllamaClient) hashText(text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", hash)
}

// GetStats returns current client statistics
func (c *OllamaClient) GetStats() Stats {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()

	return Stats{
		TotalRequests: c.stats.TotalRequests,
		TotalTokens:   c.stats.TotalTokens,
		CacheHits:     c.stats.CacheHits,
		CacheMisses:   c.stats.CacheMisses,
		Errors:        c.stats.Errors,
		RetryCount:    c.stats.RetryCount,
	}
}

// CacheSize returns the number of entries in the cache
func (c *OllamaClient) CacheSize() int {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	return len(c.cache)
}

// ClearCache clears the embedding cache
func (c *OllamaClient) ClearCache() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cache = make(map[string]*CacheEntry)
}
