package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// MetricsHandler provides endpoints for monitoring and metrics
type MetricsHandler struct {
	enhancedChat *EnhancedChatHandler
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(enhancedChat *EnhancedChatHandler) *MetricsHandler {
	return &MetricsHandler{
		enhancedChat: enhancedChat,
	}
}

// MetricsResponse represents the overall metrics response
type MetricsResponse struct {
	Timestamp       time.Time                   `json:"timestamp"`
	ProviderMetrics map[string]*ProviderMetrics `json:"provider_metrics"`
	CacheStats      map[string]interface{}      `json:"cache_stats"`
	SystemInfo      map[string]interface{}      `json:"system_info"`
}

// HandleMetrics returns comprehensive metrics data
func (mh *MetricsHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := MetricsResponse{
		Timestamp:       time.Now(),
		ProviderMetrics: mh.enhancedChat.GetMetrics(),
		CacheStats:      mh.enhancedChat.GetCacheStats(),
		SystemInfo: map[string]interface{}{
			"uptime":  time.Since(time.Now()).String(), // This would be calculated from service start time
			"version": "enhanced-v1.0",
			"features": []string{
				"retry_logic",
				"circuit_breaker",
				"request_caching",
				"context_management",
				"performance_monitoring",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
	}
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status     string                 `json:"status"`
	Timestamp  time.Time              `json:"timestamp"`
	Version    string                 `json:"version"`
	Providers  map[string]interface{} `json:"providers"`
	Components map[string]interface{} `json:"components"`
}

// HandleHealth provides a health check endpoint
func (mh *MetricsHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check component health
	components := make(map[string]interface{})

	// Cache health
	cacheStats := mh.enhancedChat.GetCacheStats()
	components["cache"] = map[string]interface{}{
		"status":      "healthy",
		"enabled":     cacheStats["enabled"],
		"entry_count": cacheStats["entry_count"],
	}

	// Metrics health
	providerMetrics := mh.enhancedChat.GetMetrics()
	totalRequests := int64(0)
	totalErrors := int64(0)

	for _, metrics := range providerMetrics {
		totalRequests += metrics.RequestCount
		totalErrors += metrics.ErrorCount
	}

	errorRate := float64(0)
	if totalRequests > 0 {
		errorRate = float64(totalErrors) / float64(totalRequests) * 100
	}

	components["metrics"] = map[string]interface{}{
		"status":         "healthy",
		"total_requests": totalRequests,
		"error_rate":     errorRate,
	}

	// Provider status
	providers := make(map[string]interface{})
	for name, metrics := range providerMetrics {
		status := "healthy"
		if metrics.RequestCount > 0 {
			providerErrorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount) * 100
			if providerErrorRate > 50 {
				status = "unhealthy"
			} else if providerErrorRate > 20 {
				status = "degraded"
			}
		}

		providers[name] = map[string]interface{}{
			"status":       status,
			"requests":     metrics.RequestCount,
			"errors":       metrics.ErrorCount,
			"cache_hits":   metrics.CacheHits,
			"cache_misses": metrics.CacheMisses,
			"retries":      metrics.RetryAttempts,
		}
	}

	overallStatus := "healthy"
	if errorRate > 50 {
		overallStatus = "unhealthy"
	} else if errorRate > 20 {
		overallStatus = "degraded"
	}

	response := HealthResponse{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Version:    "enhanced-v1.0",
		Providers:  providers,
		Components: components,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode health response", http.StatusInternalServerError)
	}
}

// HandleCacheClear provides an endpoint to clear the cache
func (mh *MetricsHandler) HandleCacheClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clear cache through the enhanced chat handler
	mh.enhancedChat.cache.Clear()

	response := map[string]interface{}{
		"status":    "success",
		"message":   "Cache cleared successfully",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Clear method for ResponseCache
func (rc *ResponseCache) Clear() {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	rc.entries = make(map[string]*CacheEntry)
}
