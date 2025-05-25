package metrics

import (
	"fmt"
	"sync"
	"time"
)

// ProcessingMetrics tracks real-time performance metrics for indexing
type ProcessingMetrics struct {
	mu sync.RWMutex

	// Timing
	startTime      time.Time
	lastUpdateTime time.Time

	// Counters
	pagesProcessed    int
	topicsProcessed   int
	uniqueTopicsFound int

	// Rates (per minute)
	currentPageRate  float64
	currentTopicRate float64
	averagePageRate  float64
	averageTopicRate float64

	// Total expected items (for ETC calculation)
	expectedPages  int
	expectedTopics int
}

// NewTracker creates a new metrics tracker for a sub-forum indexing operation
func NewTracker(expectedPages, expectedTopics int) *ProcessingMetrics {
	now := time.Now()
	return &ProcessingMetrics{
		startTime:      now,
		lastUpdateTime: now,
		expectedPages:  expectedPages,
		expectedTopics: expectedTopics,
	}
}

// UpdateMetrics updates the metrics with new processing data
func (m *ProcessingMetrics) UpdateMetrics(pagesProcessed, topicsProcessed, uniqueTopicsFound int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	timeElapsed := now.Sub(m.lastUpdateTime).Minutes()
	if timeElapsed < 0.1 { // Prevent division by very small numbers
		timeElapsed = 0.1
	}

	// Update counters
	m.pagesProcessed = pagesProcessed
	m.topicsProcessed = topicsProcessed
	m.uniqueTopicsFound = uniqueTopicsFound

	// Calculate current rates
	m.currentPageRate = float64(pagesProcessed) / timeElapsed
	m.currentTopicRate = float64(topicsProcessed) / timeElapsed

	// Calculate average rates
	totalTimeElapsed := now.Sub(m.startTime).Minutes()
	if totalTimeElapsed < 0.1 {
		totalTimeElapsed = 0.1
	}
	m.averagePageRate = float64(pagesProcessed) / totalTimeElapsed
	m.averageTopicRate = float64(topicsProcessed) / totalTimeElapsed

	m.lastUpdateTime = now
}

// GetETC calculates and returns the estimated time to completion
func (m *ProcessingMetrics) GetETC() (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.expectedPages == 0 || m.expectedTopics == 0 {
		return 0, fmt.Errorf("expected counts not set")
	}

	// Use the average rate for more stable ETC
	remainingPages := m.expectedPages - m.pagesProcessed
	if remainingPages <= 0 {
		return 0, nil
	}

	// Calculate ETC based on pages (more reliable than topics)
	etcMinutes := float64(remainingPages) / m.averagePageRate
	return time.Duration(etcMinutes * float64(time.Minute)), nil
}

// GetCurrentRates returns the current processing rates
func (m *ProcessingMetrics) GetCurrentRates() (pagesPerMin, topicsPerMin float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentPageRate, m.currentTopicRate
}

// GetAverageRates returns the average processing rates
func (m *ProcessingMetrics) GetAverageRates() (pagesPerMin, topicsPerMin float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.averagePageRate, m.averageTopicRate
}

// GetProgress returns the current progress as a percentage
func (m *ProcessingMetrics) GetProgress() (pageProgress, topicProgress float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.expectedPages > 0 {
		pageProgress = float64(m.pagesProcessed) / float64(m.expectedPages) * 100
	}
	if m.expectedTopics > 0 {
		topicProgress = float64(m.topicsProcessed) / float64(m.expectedTopics) * 100
	}
	return pageProgress, topicProgress
}

// GetElapsedTime returns the total elapsed time since tracking began
func (m *ProcessingMetrics) GetElapsedTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.startTime)
}

// GetUniqueTopicsCount returns the number of unique topics found
func (m *ProcessingMetrics) GetUniqueTopicsCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.uniqueTopicsFound
}
