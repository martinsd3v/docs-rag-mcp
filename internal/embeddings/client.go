package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"golang.org/x/time/rate"
)

const (
	// DefaultModel is the default embedding model to use
	DefaultModel = "text-embedding-3-small"

	// EmbeddingDimension is the dimension of text-embedding-3-small vectors
	EmbeddingDimension = 1536

	// DefaultRateLimit is the default rate limit (requests per minute)
	DefaultRateLimit = 1000

	// DefaultBatchSize is the default batch size for embedding requests
	DefaultBatchSize = 100

	// DefaultMaxRetries is the default number of retries for failed requests
	DefaultMaxRetries = 3
)

// ClientConfig holds configuration for the OpenAI client
type ClientConfig struct {
	APIKey       string        `json:"api_key"`
	BaseURL      string        `json:"base_url"`      // custom API endpoint (optional)
	Model        string        `json:"model"`
	RateLimit    int           `json:"rate_limit"`    // requests per minute
	BatchSize    int           `json:"batch_size"`    // texts per batch
	MaxRetries   int           `json:"max_retries"`   // max retry attempts
	Timeout      time.Duration `json:"timeout"`       // request timeout
	RetryBackoff []time.Duration `json:"retry_backoff"` // backoff durations
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig(apiKey string) *ClientConfig {
	return &ClientConfig{
		APIKey:     apiKey,
		BaseURL:    "", // empty means use default OpenAI endpoint
		Model:      DefaultModel,
		RateLimit:  DefaultRateLimit,
		BatchSize:  DefaultBatchSize,
		MaxRetries: DefaultMaxRetries,
		Timeout:    30 * time.Second,
		RetryBackoff: []time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
		},
	}
}

// EmbeddingResult holds the result of an embedding request
type EmbeddingResult struct {
	Text       string    `json:"text"`
	Embedding  []float32 `json:"embedding"`
	TokenCount int       `json:"token_count"`
	Cached     bool      `json:"cached"`
}

// BatchResult holds the results of a batch embedding request
type BatchResult struct {
	Results    []EmbeddingResult `json:"results"`
	TotalTokens int              `json:"total_tokens"`
	CacheHits   int              `json:"cache_hits"`
	CacheMisses int              `json:"cache_misses"`
	Duration    time.Duration    `json:"duration"`
}

// CacheEntry represents a cached embedding
type CacheEntry struct {
	Embedding  []float32
	TokenCount int
	Model      string
	CreatedAt  time.Time
	LastUsed   time.Time
	UseCount   int
}

// Client wraps the OpenAI API for embedding generation
type Client struct {
	client     *openai.Client
	config     *ClientConfig
	limiter    *rate.Limiter
	cache      map[string]*CacheEntry
	cacheMutex sync.RWMutex
	stats      *Stats
}

// Stats tracks client statistics
type Stats struct {
	TotalRequests   int64
	TotalTokens     int64
	CacheHits       int64
	CacheMisses     int64
	Errors          int64
	RetryCount      int64
	mutex           sync.Mutex
}

// NewClient creates a new OpenAI embedding client
func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Set defaults if not specified
	if config.Model == "" {
		config.Model = DefaultModel
	}
	if config.RateLimit == 0 {
		config.RateLimit = DefaultRateLimit
	}
	if config.BatchSize == 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = DefaultMaxRetries
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if len(config.RetryBackoff) == 0 {
		config.RetryBackoff = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	}

	// Create OpenAI client with options
	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}

	// Add custom base URL if provided
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	client := openai.NewClient(opts...)

	// Create rate limiter (requests per second)
	rps := float64(config.RateLimit) / 60.0
	limiter := rate.NewLimiter(rate.Limit(rps), config.BatchSize)

	return &Client{
		client:  &client,
		config:  config,
		limiter: limiter,
		cache:   make(map[string]*CacheEntry),
		stats:   &Stats{},
	}, nil
}

// Embed generates an embedding for a single text
func (c *Client) Embed(ctx context.Context, text string) (*EmbeddingResult, error) {
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
func (c *Client) EmbedBatch(ctx context.Context, texts []string) (*BatchResult, error) {
	start := time.Now()
	result := &BatchResult{
		Results: make([]EmbeddingResult, len(texts)),
	}

	// Check cache first
	textsToEmbed := make([]string, 0)
	textIndices := make([]int, 0)

	for i, text := range texts {
		// Skip empty strings - they cause 422 errors from the API
		if strings.TrimSpace(text) == "" {
			result.Results[i] = EmbeddingResult{
				Text:       text,
				Embedding:  make([]float32, EmbeddingDimension), // zero vector for empty text
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
		embeddings, tokens, err := c.generateEmbeddings(ctx, textsToEmbed)
		if err != nil {
			return nil, err
		}

		// Store results and update cache
		for i, embedding := range embeddings {
			idx := textIndices[i]
			text := textsToEmbed[i]
			tokenCount := 0
			if i < len(tokens) {
				tokenCount = tokens[i]
			}

			result.Results[idx] = EmbeddingResult{
				Text:       text,
				Embedding:  embedding,
				TokenCount: tokenCount,
				Cached:     false,
			}

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

		// Update total tokens
		for _, t := range tokens {
			result.TotalTokens += t
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// generateEmbeddings calls the OpenAI API to generate embeddings
func (c *Client) generateEmbeddings(ctx context.Context, texts []string) ([][]float32, []int, error) {
	allEmbeddings := make([][]float32, 0, len(texts))
	allTokens := make([]int, 0, len(texts))

	// Process in batches
	for i := 0; i < len(texts); i += c.config.BatchSize {
		end := i + c.config.BatchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		embeddings, tokens, err := c.embedBatchWithRetry(ctx, batch)
		if err != nil {
			return nil, nil, fmt.Errorf("batch %d failed: %w", i/c.config.BatchSize, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
		allTokens = append(allTokens, tokens...)
	}

	return allEmbeddings, allTokens, nil
}

// embedBatchWithRetry calls OpenAI API with retry logic
func (c *Client) embedBatchWithRetry(ctx context.Context, texts []string) ([][]float32, []int, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Wait for rate limiter
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, nil, fmt.Errorf("rate limiter error: %w", err)
		}

		embeddings, tokens, err := c.callOpenAI(ctx, texts)
		if err == nil {
			return embeddings, tokens, nil
		}

		lastErr = err

		// Check if error is retryable
		if !c.isRetryableError(err) {
			return nil, nil, err
		}

		// Wait before retry
		if attempt < c.config.MaxRetries {
			backoff := c.config.RetryBackoff[min(attempt, len(c.config.RetryBackoff)-1)]
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
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

	return nil, nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// callOpenAI makes the actual API call
func (c *Client) callOpenAI(ctx context.Context, texts []string) ([][]float32, []int, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Build input for API call
	input := make([]string, len(texts))
	copy(input, texts)

	// Call OpenAI API
	resp, err := c.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: input,
		},
		Model: openai.EmbeddingModel(c.config.Model),
	})
	if err != nil {
		return nil, nil, err
	}

	c.stats.mutex.Lock()
	c.stats.TotalRequests++
	c.stats.TotalTokens += int64(resp.Usage.TotalTokens)
	c.stats.mutex.Unlock()

	// Extract embeddings
	embeddings := make([][]float32, len(resp.Data))
	tokens := make([]int, len(resp.Data))

	for _, data := range resp.Data {
		idx := data.Index
		if idx >= int64(len(embeddings)) {
			continue
		}

		// Convert float64 to float32
		embedding := make([]float32, len(data.Embedding))
		for i, v := range data.Embedding {
			embedding[i] = float32(v)
		}
		embeddings[idx] = embedding

		// Estimate tokens per text (evenly distributed)
		tokens[idx] = int(resp.Usage.TotalTokens) / len(texts)
	}

	return embeddings, tokens, nil
}

// isRetryableError checks if an error should be retried
func (c *Client) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Rate limit errors (429)
	if contains(errStr, "rate_limit") || contains(errStr, "429") {
		return true
	}

	// Server errors (5xx)
	if contains(errStr, "500") || contains(errStr, "502") ||
		contains(errStr, "503") || contains(errStr, "504") {
		return true
	}

	// Timeout errors
	if contains(errStr, "timeout") || contains(errStr, "deadline") {
		return true
	}

	return false
}

// hashText creates a SHA256 hash of the text for caching
func (c *Client) hashText(text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", hash)
}

// GetStats returns current client statistics
func (c *Client) GetStats() Stats {
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
func (c *Client) CacheSize() int {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	return len(c.cache)
}

// ClearCache clears the embedding cache
func (c *Client) ClearCache() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cache = make(map[string]*CacheEntry)
}

// CleanCache removes old entries from cache
func (c *Client) CleanCache(maxAge time.Duration, maxEntries int) int {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	now := time.Now()
	removed := 0

	// Remove old entries
	for hash, entry := range c.cache {
		if now.Sub(entry.LastUsed) > maxAge {
			delete(c.cache, hash)
			removed++
		}
	}

	// If still too many entries, remove least recently used
	if len(c.cache) > maxEntries {
		// Build list sorted by last used
		type cacheItem struct {
			hash     string
			lastUsed time.Time
		}
		items := make([]cacheItem, 0, len(c.cache))
		for hash, entry := range c.cache {
			items = append(items, cacheItem{hash: hash, lastUsed: entry.LastUsed})
		}

		// Simple sort by last used (oldest first)
		for i := 0; i < len(items)-1; i++ {
			for j := i + 1; j < len(items); j++ {
				if items[j].lastUsed.Before(items[i].lastUsed) {
					items[i], items[j] = items[j], items[i]
				}
			}
		}

		// Remove oldest entries
		toRemove := len(c.cache) - maxEntries
		for i := 0; i < toRemove && i < len(items); i++ {
			delete(c.cache, items[i].hash)
			removed++
		}
	}

	return removed
}

// GetCacheEntry retrieves a cache entry by text hash
func (c *Client) GetCacheEntry(text string) (*CacheEntry, bool) {
	hash := c.hashText(text)
	c.cacheMutex.RLock()
	entry, found := c.cache[hash]
	c.cacheMutex.RUnlock()
	return entry, found
}

// SetCacheEntry manually adds an entry to the cache
func (c *Client) SetCacheEntry(text string, embedding []float32, tokenCount int) {
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

// CosineSimilarity calculates cosine similarity between two embeddings
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// EmbeddingToBytes converts a float32 embedding to bytes for storage
func EmbeddingToBytes(embedding []float32) []byte {
	buf := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// BytesToEmbedding converts bytes back to float32 embedding
func BytesToEmbedding(data []byte) []float32 {
	embedding := make([]float32, len(data)/4)
	for i := range embedding {
		embedding[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return embedding
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
