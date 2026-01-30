package embeddings

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig("test-key")

	assert.Equal(t, "test-key", config.APIKey)
	assert.Equal(t, DefaultModel, config.Model)
	assert.Equal(t, DefaultRateLimit, config.RateLimit)
	assert.Equal(t, DefaultBatchSize, config.BatchSize)
	assert.Equal(t, DefaultMaxRetries, config.MaxRetries)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Len(t, config.RetryBackoff, 3)
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *ClientConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty API key",
			config: &ClientConfig{
				APIKey: "",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &ClientConfig{
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name:    "default config",
			config:  DefaultClientConfig("test-key"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_Cache(t *testing.T) {
	config := DefaultClientConfig("test-key")
	client, err := NewClient(config)
	require.NoError(t, err)

	// Test cache operations
	text := "test text for caching"
	embedding := make([]float32, EmbeddingDimension)
	for i := range embedding {
		embedding[i] = float32(i) / float32(EmbeddingDimension)
	}

	// Cache should be empty initially
	assert.Equal(t, 0, client.CacheSize())

	// Add to cache
	client.SetCacheEntry(text, embedding, 5)
	assert.Equal(t, 1, client.CacheSize())

	// Retrieve from cache
	entry, found := client.GetCacheEntry(text)
	assert.True(t, found)
	assert.Equal(t, embedding, entry.Embedding)
	assert.Equal(t, 5, entry.TokenCount)
	assert.Equal(t, config.Model, entry.Model)

	// Different text should not be found
	_, found = client.GetCacheEntry("different text")
	assert.False(t, found)

	// Clear cache
	client.ClearCache()
	assert.Equal(t, 0, client.CacheSize())
}

func TestClient_CleanCache(t *testing.T) {
	config := DefaultClientConfig("test-key")
	client, err := NewClient(config)
	require.NoError(t, err)

	embedding := make([]float32, EmbeddingDimension)

	// Add multiple entries
	for i := 0; i < 10; i++ {
		client.SetCacheEntry(string(rune('a'+i)), embedding, 1)
	}
	assert.Equal(t, 10, client.CacheSize())

	// Clean with max entries limit
	removed := client.CleanCache(24*time.Hour, 5)
	assert.Equal(t, 5, removed)
	assert.Equal(t, 5, client.CacheSize())
}

func TestClient_Stats(t *testing.T) {
	config := DefaultClientConfig("test-key")
	client, err := NewClient(config)
	require.NoError(t, err)

	stats := client.GetStats()
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.CacheHits)
	assert.Equal(t, int64(0), stats.CacheMisses)
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "different lengths",
			a:        []float32{1, 0},
			b:        []float32{1, 0, 0},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 0.0,
		},
		{
			name:     "similar vectors",
			a:        []float32{1, 1, 0},
			b:        []float32{1, 0, 0},
			expected: 0.7071067811865476, // 1/sqrt(2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestEmbeddingToBytes(t *testing.T) {
	original := []float32{1.5, -2.5, 3.14159, 0.0}

	// Convert to bytes
	bytes := EmbeddingToBytes(original)
	assert.Len(t, bytes, len(original)*4)

	// Convert back
	restored := BytesToEmbedding(bytes)
	assert.Equal(t, len(original), len(restored))

	for i, v := range original {
		assert.InDelta(t, v, restored[i], 0.00001)
	}
}

func TestClient_HashText(t *testing.T) {
	config := DefaultClientConfig("test-key")
	client, err := NewClient(config)
	require.NoError(t, err)

	// Same text should produce same hash
	hash1 := client.hashText("test text")
	hash2 := client.hashText("test text")
	assert.Equal(t, hash1, hash2)

	// Different text should produce different hash
	hash3 := client.hashText("different text")
	assert.NotEqual(t, hash1, hash3)

	// Hash should be 64 characters (SHA256 hex)
	assert.Len(t, hash1, 64)
}

func TestClient_IsRetryableError(t *testing.T) {
	config := DefaultClientConfig("test-key")
	client, err := NewClient(config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "rate limit error",
			err:      fmt.Errorf("rate_limit exceeded"),
			expected: true,
		},
		{
			name:     "429 error",
			err:      fmt.Errorf("HTTP 429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "500 error",
			err:      fmt.Errorf("HTTP 500 Internal Server Error"),
			expected: true,
		},
		{
			name:     "503 error",
			err:      fmt.Errorf("HTTP 503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      fmt.Errorf("request timeout"),
			expected: true,
		},
		{
			name:     "deadline error",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      fmt.Errorf("invalid API key"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark for cosine similarity
func BenchmarkCosineSimilarity(b *testing.B) {
	a := make([]float32, EmbeddingDimension)
	bVec := make([]float32, EmbeddingDimension)

	for i := range a {
		a[i] = float32(i) / float32(EmbeddingDimension)
		bVec[i] = float32(EmbeddingDimension-i) / float32(EmbeddingDimension)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(a, bVec)
	}
}

// Benchmark for embedding serialization
func BenchmarkEmbeddingToBytes(b *testing.B) {
	embedding := make([]float32, EmbeddingDimension)
	for i := range embedding {
		embedding[i] = float32(i) / float32(EmbeddingDimension)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EmbeddingToBytes(embedding)
	}
}

func BenchmarkBytesToEmbedding(b *testing.B) {
	embedding := make([]float32, EmbeddingDimension)
	for i := range embedding {
		embedding[i] = float32(i) / float32(EmbeddingDimension)
	}
	bytes := EmbeddingToBytes(embedding)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BytesToEmbedding(bytes)
	}
}
