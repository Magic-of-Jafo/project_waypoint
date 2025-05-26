package metrics

import (
	"time"
)

// BatchMetrics tracks real-time metrics for the current archival batch
type BatchMetrics struct {
	// Start time of the current batch
	StartTime time.Time

	// Counters
	PagesArchived  int64
	TopicsArchived int64
	BytesArchived  int64

	// Current rates (updated periodically)
	CurrentPagesPerMin   float64
	CurrentTopicsPerHour float64
	CurrentMBPerMin      float64

	// For rate smoothing
	lastUpdateTime time.Time
	lastPages      int64
	lastTopics     int64
	lastBytes      int64
}

// HistoricalMetrics represents a single completed batch's performance data
type HistoricalMetrics struct {
	TimestampUTC     time.Time
	BatchID          string
	DurationSeconds  float64
	PagesArchived    int64
	TopicsArchived   int64
	BytesArchived    int64
	AvgPagesPerMin   float64
	AvgTopicsPerHour float64
	AvgMBPerMin      float64
}

// NewBatchMetrics creates a new BatchMetrics instance with initialized start time
func NewBatchMetrics() *BatchMetrics {
	now := time.Now()
	return &BatchMetrics{
		StartTime:      now,
		lastUpdateTime: now,
	}
}

// UpdateRates calculates current processing rates based on elapsed time
func (m *BatchMetrics) UpdateRates() {
	now := time.Now()
	elapsed := now.Sub(m.lastUpdateTime).Minutes()

	if elapsed > 0 {
		// Calculate rates based on delta since last update
		deltaPages := m.PagesArchived - m.lastPages
		deltaTopics := m.TopicsArchived - m.lastTopics
		deltaBytes := m.BytesArchived - m.lastBytes

		m.CurrentPagesPerMin = float64(deltaPages) / elapsed
		m.CurrentTopicsPerHour = float64(deltaTopics) / (elapsed / 60)
		m.CurrentMBPerMin = float64(deltaBytes) / (elapsed * 1024 * 1024)

		// Update last values
		m.lastPages = m.PagesArchived
		m.lastTopics = m.TopicsArchived
		m.lastBytes = m.BytesArchived
		m.lastUpdateTime = now
	}
}

// GetETC calculates estimated time to completion based on current rates and remaining items
func (m *BatchMetrics) GetETC(remainingPages, remainingTopics int64) time.Duration {
	// Use the slower rate between pages and topics to be conservative
	var pagesETC, topicsETC time.Duration

	if m.CurrentPagesPerMin > 0 {
		pagesETC = time.Duration(float64(remainingPages)/m.CurrentPagesPerMin) * time.Minute
	}
	if m.CurrentTopicsPerHour > 0 {
		topicsETC = time.Duration(float64(remainingTopics)/m.CurrentTopicsPerHour) * time.Hour
	}

	// Return the longer estimate
	if pagesETC > topicsETC {
		return pagesETC
	}
	return topicsETC
}

// ToHistoricalMetrics converts current batch metrics to historical format
func (m *BatchMetrics) ToHistoricalMetrics(batchID string) HistoricalMetrics {
	duration := time.Since(m.StartTime).Seconds()

	return HistoricalMetrics{
		TimestampUTC:     time.Now().UTC(),
		BatchID:          batchID,
		DurationSeconds:  duration,
		PagesArchived:    m.PagesArchived,
		TopicsArchived:   m.TopicsArchived,
		BytesArchived:    m.BytesArchived,
		AvgPagesPerMin:   float64(m.PagesArchived) / (duration / 60),
		AvgTopicsPerHour: float64(m.TopicsArchived) / (duration / 3600),
		AvgMBPerMin:      float64(m.BytesArchived) / (duration * 1024 * 1024),
	}
}
