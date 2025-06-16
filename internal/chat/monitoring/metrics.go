package monitoring

import (
	"sync"
	"time"

	"github.com/yourusername/gogdbllm/internal/chat"
)

// MetricsCollector handles metrics collection and aggregation
type MetricsCollector struct {
	providerMetrics map[string]*chat.Metrics
	globalMetrics   *chat.Metrics
	mutex           sync.RWMutex
	startTime       time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		providerMetrics: make(map[string]*chat.Metrics),
		globalMetrics:   &chat.Metrics{},
		startTime:       time.Now(),
	}
}

// RecordRequest records a request metric
func (mc *MetricsCollector) RecordRequest(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.providerMetrics[provider] == nil {
		mc.providerMetrics[provider] = &chat.Metrics{}
	}

	mc.providerMetrics[provider].RequestCount++
	mc.globalMetrics.RequestCount++
}

// RecordResponse records a response metric
func (mc *MetricsCollector) RecordResponse(provider string, responseTime time.Duration, tokensUsed int, cost float64) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.providerMetrics[provider] == nil {
		mc.providerMetrics[provider] = &chat.Metrics{}
	}

	providerMetrics := mc.providerMetrics[provider]

	// Update response time (running average)
	if providerMetrics.RequestCount > 0 {
		providerMetrics.ResponseTime = time.Duration(
			(int64(providerMetrics.ResponseTime) + int64(responseTime)) / 2,
		)
	} else {
		providerMetrics.ResponseTime = responseTime
	}

	providerMetrics.TokensUsed += int64(tokensUsed)
	providerMetrics.EstimatedCost += cost

	// Update global metrics
	if mc.globalMetrics.RequestCount > 0 {
		mc.globalMetrics.ResponseTime = time.Duration(
			(int64(mc.globalMetrics.ResponseTime) + int64(responseTime)) / 2,
		)
	} else {
		mc.globalMetrics.ResponseTime = responseTime
	}

	mc.globalMetrics.TokensUsed += int64(tokensUsed)
	mc.globalMetrics.EstimatedCost += cost
}

// RecordError records an error metric
func (mc *MetricsCollector) RecordError(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.providerMetrics[provider] == nil {
		mc.providerMetrics[provider] = &chat.Metrics{}
	}

	mc.providerMetrics[provider].ErrorCount++
	mc.globalMetrics.ErrorCount++
}

// RecordCacheHit records a cache hit
func (mc *MetricsCollector) RecordCacheHit(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.providerMetrics[provider] == nil {
		mc.providerMetrics[provider] = &chat.Metrics{}
	}

	mc.providerMetrics[provider].CacheHits++
	mc.globalMetrics.CacheHits++
}

// RecordCacheMiss records a cache miss
func (mc *MetricsCollector) RecordCacheMiss(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.providerMetrics[provider] == nil {
		mc.providerMetrics[provider] = &chat.Metrics{}
	}

	mc.providerMetrics[provider].CacheMisses++
	mc.globalMetrics.CacheMisses++
}

// RecordRetry records a retry attempt
func (mc *MetricsCollector) RecordRetry(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.providerMetrics[provider] == nil {
		mc.providerMetrics[provider] = &chat.Metrics{}
	}

	mc.providerMetrics[provider].RetryAttempts++
	mc.globalMetrics.RetryAttempts++
}

// RecordCircuitBreakerTrip records a circuit breaker trip
func (mc *MetricsCollector) RecordCircuitBreakerTrip(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.providerMetrics[provider] == nil {
		mc.providerMetrics[provider] = &chat.Metrics{}
	}

	mc.providerMetrics[provider].CircuitBreakerTrips++
	mc.globalMetrics.CircuitBreakerTrips++
}

// RecordContextTrim records a context trimming event
func (mc *MetricsCollector) RecordContextTrim(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.providerMetrics[provider] == nil {
		mc.providerMetrics[provider] = &chat.Metrics{}
	}

	mc.providerMetrics[provider].ContextTrimCount++
	mc.globalMetrics.ContextTrimCount++
}

// GetProviderMetrics returns metrics for a specific provider
func (mc *MetricsCollector) GetProviderMetrics(provider string) *chat.ProviderMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	metrics := mc.providerMetrics[provider]
	if metrics == nil {
		metrics = &chat.Metrics{}
	}

	// Create a copy to avoid data races
	metricsCopy := *metrics

	return &chat.ProviderMetrics{
		Provider:    provider,
		Metrics:     &metricsCopy,
		LastUpdated: time.Now(),
	}
}

// GetGlobalMetrics returns global metrics
func (mc *MetricsCollector) GetGlobalMetrics() *chat.Metrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// Create a copy to avoid data races
	metricsCopy := *mc.globalMetrics
	return &metricsCopy
}

// GetAllProviderMetrics returns metrics for all providers
func (mc *MetricsCollector) GetAllProviderMetrics() map[string]*chat.ProviderMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	result := make(map[string]*chat.ProviderMetrics)
	for provider := range mc.providerMetrics {
		result[provider] = mc.GetProviderMetrics(provider)
	}

	return result
}

// GetErrorRate returns the error rate for a provider
func (mc *MetricsCollector) GetErrorRate(provider string) float64 {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	metrics := mc.providerMetrics[provider]
	if metrics == nil || metrics.RequestCount == 0 {
		return 0.0
	}

	return float64(metrics.ErrorCount) / float64(metrics.RequestCount) * 100
}

// GetCacheHitRate returns the cache hit rate for a provider
func (mc *MetricsCollector) GetCacheHitRate(provider string) float64 {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	metrics := mc.providerMetrics[provider]
	if metrics == nil {
		return 0.0
	}

	totalCacheRequests := metrics.CacheHits + metrics.CacheMisses
	if totalCacheRequests == 0 {
		return 0.0
	}

	return float64(metrics.CacheHits) / float64(totalCacheRequests) * 100
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.providerMetrics = make(map[string]*chat.Metrics)
	mc.globalMetrics = &chat.Metrics{}
	mc.startTime = time.Now()
}

// GetUptime returns the uptime of the metrics collector
func (mc *MetricsCollector) GetUptime() time.Duration {
	return time.Since(mc.startTime)
}
