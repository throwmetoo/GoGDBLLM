package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/gogdbllm/internal/chat"
)

// Config holds cache configuration
type Config struct {
	Enabled     bool          `yaml:"enabled"`
	TTL         time.Duration `yaml:"ttl"`
	MaxSize     int           `yaml:"max_size"`
	Compression bool          `yaml:"compression"`
}

// DefaultConfig returns default cache configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:     false,
		TTL:         time.Hour,
		MaxSize:     1000,
		Compression: true,
	}
}

// Cache represents an in-memory cache for chat requests/responses
type Cache struct {
	config      *Config
	entries     map[string]*chat.CacheEntry
	accessOrder []string // For LRU eviction
	mutex       sync.RWMutex
	stats       *CacheStats
}

// CacheStats holds cache performance statistics
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	Evictions   int64   `json:"evictions"`
	Size        int     `json:"size"`
	HitRate     float64 `json:"hit_rate"`
	MemoryUsage int64   `json:"memory_usage_bytes"`
}

// New creates a new cache instance
func New(config *Config) *Cache {
	if config == nil {
		config = DefaultConfig()
	}

	return &Cache{
		config:      config,
		entries:     make(map[string]*chat.CacheEntry),
		accessOrder: make([]string, 0),
		stats:       &CacheStats{},
	}
}

// Get retrieves a cached response
func (c *Cache) Get(key *chat.CacheKey) (*chat.ChatResponse, bool) {
	if !c.config.Enabled {
		return nil, false
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	keyStr := c.keyToString(key)
	entry, exists := c.entries[keyStr]

	if !exists {
		c.stats.Misses++
		c.updateStats()
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, keyStr)
		c.removeFromAccessOrder(keyStr)
		c.stats.Misses++
		c.updateStats()
		return nil, false
	}

	// Update access information
	entry.AccessCount++
	entry.LastAccessed = time.Now()
	c.moveToFront(keyStr)

	c.stats.Hits++
	c.updateStats()

	// Mark response as from cache
	response := *entry.Response
	response.FromCache = true

	return &response, true
}

// Set stores a response in the cache
func (c *Cache) Set(key *chat.CacheKey, response *chat.ChatResponse) {
	if !c.config.Enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	keyStr := c.keyToString(key)

	// Create cache entry
	entry := &chat.CacheEntry{
		Response:     response,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(c.config.TTL),
		AccessCount:  1,
		LastAccessed: time.Now(),
	}

	// Check if we need to evict entries
	if len(c.entries) >= c.config.MaxSize {
		c.evictLRU()
	}

	// Store the entry
	c.entries[keyStr] = entry
	c.moveToFront(keyStr)

	c.updateStats()
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[string]*chat.CacheEntry)
	c.accessOrder = make([]string, 0)
	c.stats = &CacheStats{}
}

// GetStats returns cache statistics
func (c *Cache) GetStats() *CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	statsCopy := *c.stats
	return &statsCopy
}

// IsEnabled returns whether caching is enabled
func (c *Cache) IsEnabled() bool {
	return c.config.Enabled
}

// keyToString converts a cache key to a string representation
func (c *Cache) keyToString(key *chat.CacheKey) string {
	if key.Hash != "" {
		return fmt.Sprintf("%s:%s:%s", key.Provider, key.Model, key.Hash)
	}

	// If no hash provided, this is likely an error
	return fmt.Sprintf("%s:%s:no-hash", key.Provider, key.Model)
}

// GenerateKey generates a cache key for a request
func (c *Cache) GenerateKey(provider, model string, request *chat.ChatRequest) *chat.CacheKey {
	// Create a consistent hash of the request
	hash := c.hashRequest(request)

	return &chat.CacheKey{
		Provider: provider,
		Model:    model,
		Hash:     hash,
	}
}

// hashRequest creates a hash of the request for cache key generation
func (c *Cache) hashRequest(request *chat.ChatRequest) string {
	// Create a simplified version of the request for hashing
	hashData := struct {
		Message     string                 `json:"message"`
		History     []chat.StandardMessage `json:"history"`
		SentContext []interface{}          `json:"sentContext"`
	}{
		Message:     request.Message,
		History:     make([]chat.StandardMessage, len(request.History)),
		SentContext: make([]interface{}, len(request.SentContext)),
	}

	// Convert history to standard messages
	for i, msg := range request.History {
		hashData.History[i] = chat.StandardMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Convert context items (simplified)
	for i, ctx := range request.SentContext {
		hashData.SentContext[i] = map[string]string{
			"type":        ctx.Type,
			"description": ctx.Description,
			"content":     ctx.Content,
		}
	}

	// Marshal to JSON and hash
	data, _ := json.Marshal(hashData)
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 characters
}

// evictLRU removes the least recently used entry
func (c *Cache) evictLRU() {
	if len(c.accessOrder) == 0 {
		return
	}

	// Remove the least recently used entry (last in access order)
	keyToEvict := c.accessOrder[len(c.accessOrder)-1]
	delete(c.entries, keyToEvict)
	c.accessOrder = c.accessOrder[:len(c.accessOrder)-1]
	c.stats.Evictions++
}

// moveToFront moves a key to the front of the access order (most recently used)
func (c *Cache) moveToFront(key string) {
	// Remove from current position
	c.removeFromAccessOrder(key)

	// Add to front
	c.accessOrder = append([]string{key}, c.accessOrder...)
}

// removeFromAccessOrder removes a key from the access order slice
func (c *Cache) removeFromAccessOrder(key string) {
	for i, k := range c.accessOrder {
		if k == key {
			c.accessOrder = append(c.accessOrder[:i], c.accessOrder[i+1:]...)
			break
		}
	}
}

// updateStats updates cache statistics
func (c *Cache) updateStats() {
	c.stats.Size = len(c.entries)

	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		c.stats.HitRate = float64(c.stats.Hits) / float64(total) * 100
	}

	// Estimate memory usage (rough calculation)
	c.stats.MemoryUsage = int64(len(c.entries) * 1024) // Rough estimate: 1KB per entry
}

// Cleanup removes expired entries
func (c *Cache) Cleanup() {
	if !c.config.Enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	var keysToRemove []string

	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		delete(c.entries, key)
		c.removeFromAccessOrder(key)
	}

	c.updateStats()
}

// StartCleanupRoutine starts a background routine to clean up expired entries
func (c *Cache) StartCleanupRoutine() {
	if !c.config.Enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(c.config.TTL / 4) // Cleanup every quarter of TTL
		defer ticker.Stop()

		for range ticker.C {
			c.Cleanup()
		}
	}()
}
