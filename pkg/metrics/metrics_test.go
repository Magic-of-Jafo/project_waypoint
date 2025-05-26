package metrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBatchMetrics_UpdateRates(t *testing.T) {
	metrics := NewBatchMetrics()

	// Initial state
	if metrics.CurrentPagesPerMin != 0 || metrics.CurrentTopicsPerHour != 0 || metrics.CurrentMBPerMin != 0 {
		t.Error("Initial rates should be zero")
	}

	// Simulate some progress
	metrics.PagesArchived = 60
	metrics.TopicsArchived = 2
	metrics.BytesArchived = 1024 * 1024 // 1 MB

	// Update rates
	metrics.UpdateRates()

	// Check rates (allowing for some timing variation)
	if metrics.CurrentPagesPerMin < 0 || metrics.CurrentTopicsPerHour < 0 || metrics.CurrentMBPerMin < 0 {
		t.Error("Rates should not be negative")
	}
}

func TestBatchMetrics_GetETC(t *testing.T) {
	metrics := NewBatchMetrics()

	// Test with zero rates
	etc := metrics.GetETC(100, 10)
	if etc != 0 {
		t.Error("ETC should be zero when rates are zero")
	}

	// Set some rates
	metrics.CurrentPagesPerMin = 10
	metrics.CurrentTopicsPerHour = 1

	// Test ETC calculation
	etc = metrics.GetETC(100, 10)
	if etc == 0 {
		t.Error("ETC should be non-zero with non-zero rates")
	}

	// Test that it uses the slower rate
	etc = metrics.GetETC(100, 1)
	if etc != time.Duration(60)*time.Minute {
		t.Error("ETC should use the slower rate")
	}
}

func TestPerformanceLogger_AppendAndLoadMetrics(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "metrics_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logPath := filepath.Join(tempDir, "test_performance.csv")
	logger := NewPerformanceLogger(logPath)

	// Create test metrics
	metrics := HistoricalMetrics{
		TimestampUTC:     time.Now().UTC(),
		BatchID:          "test_batch",
		DurationSeconds:  3600,
		PagesArchived:    1800,
		TopicsArchived:   30,
		BytesArchived:    512000000,
		AvgPagesPerMin:   30.0,
		AvgTopicsPerHour: 30.0,
		AvgMBPerMin:      8.14,
	}

	// Test appending metrics
	if err := logger.AppendMetrics(metrics); err != nil {
		t.Errorf("Failed to append metrics: %v", err)
	}

	// Test loading metrics
	loadedMetrics, err := logger.LoadHistoricalMetrics()
	if err != nil {
		t.Errorf("Failed to load metrics: %v", err)
	}

	if len(loadedMetrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(loadedMetrics))
	}

	// Compare loaded metrics with original
	loaded := loadedMetrics[0]
	if loaded.BatchID != metrics.BatchID ||
		loaded.PagesArchived != metrics.PagesArchived ||
		loaded.TopicsArchived != metrics.TopicsArchived ||
		loaded.BytesArchived != metrics.BytesArchived {
		t.Error("Loaded metrics don't match original metrics")
	}
}

func TestFormatting(t *testing.T) {
	// Test duration formatting
	duration := 3*24*time.Hour + 4*time.Hour + 15*time.Minute
	expected := "3 days 4 hours 15 minutes"
	if got := FormatDuration(duration); got != expected {
		t.Errorf("FormatDuration() = %v, want %v", got, expected)
	}

	// Test byte formatting
	tests := []struct {
		bytes    int64
		expected string
	}{
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{500, "500 bytes"},
	}

	for _, tt := range tests {
		if got := FormatBytes(tt.bytes); got != tt.expected {
			t.Errorf("FormatBytes(%d) = %v, want %v", tt.bytes, got, tt.expected)
		}
	}

	// Test progress formatting
	if got := FormatProgress(1500, 11000); got != "Processed: 1500 of 11000" {
		t.Errorf("FormatProgress() = %v, want %v", got, "Processed: 1500 of 11000")
	}

	// Test rates formatting
	expected = "Rate: 30.0 pages/min, 2.0 topics/hour, 8.14 MB/min"
	if got := FormatRates(30.0, 2.0, 8.14); got != expected {
		t.Errorf("FormatRates() = %v, want %v", got, expected)
	}

	// Test ETC formatting
	expected = "ETC: ~3 days 4 hours 15 minutes"
	if got := FormatETC(duration); got != expected {
		t.Errorf("FormatETC() = %v, want %v", got, expected)
	}
}
