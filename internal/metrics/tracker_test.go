package metrics

import (
	"testing"
	"time"
)

func TestProcessingMetrics(t *testing.T) {
	// Create a new tracker with expected counts
	tracker := NewTracker(100, 1000) // 100 pages, 1000 topics expected

	// Test initial state
	if tracker.GetUniqueTopicsCount() != 0 {
		t.Errorf("Expected 0 unique topics initially, got %d", tracker.GetUniqueTopicsCount())
	}

	// Update metrics after processing some pages
	tracker.UpdateMetrics(25, 250, 245) // 25 pages, 250 topics processed, 245 unique

	// Test progress calculation
	pageProgress, topicProgress := tracker.GetProgress()
	if pageProgress != 25.0 {
		t.Errorf("Expected 25%% page progress, got %.1f%%", pageProgress)
	}
	if topicProgress != 25.0 {
		t.Errorf("Expected 25%% topic progress, got %.1f%%", topicProgress)
	}

	// Test unique topics count
	if tracker.GetUniqueTopicsCount() != 245 {
		t.Errorf("Expected 245 unique topics, got %d", tracker.GetUniqueTopicsCount())
	}

	// Test ETC calculation
	etc, err := tracker.GetETC()
	if err != nil {
		t.Errorf("Unexpected error getting ETC: %v", err)
	}
	if etc <= 0 {
		t.Errorf("Expected positive ETC, got %v", etc)
	}

	// Test rate calculations
	pageRate, topicRate := tracker.GetCurrentRates()
	if pageRate <= 0 {
		t.Errorf("Expected positive page rate, got %.2f", pageRate)
	}
	if topicRate <= 0 {
		t.Errorf("Expected positive topic rate, got %.2f", topicRate)
	}
}

func TestFormatting(t *testing.T) {
	// Test duration formatting
	durations := []struct {
		d        time.Duration
		expected string
	}{
		{30 * time.Second, "30 seconds"},
		{90 * time.Second, "1 minutes"},
		{2 * time.Hour, "2.0 hours"},
		{25 * time.Hour, "1.0 days"},
	}

	for _, tc := range durations {
		got := FormatDuration(tc.d)
		if got != tc.expected {
			t.Errorf("FormatDuration(%v) = %q, want %q", tc.d, got, tc.expected)
		}
	}

	// Test rate formatting
	rates := []struct {
		rate     float64
		unit     string
		expected string
	}{
		{0.5, "pages", "0.50 pages/minute"},
		{15.0, "topics", "15 topics/minute"},
	}

	for _, tc := range rates {
		got := FormatRate(tc.rate, tc.unit)
		if got != tc.expected {
			t.Errorf("FormatRate(%.2f, %q) = %q, want %q", tc.rate, tc.unit, got, tc.expected)
		}
	}

	// Test progress formatting
	if got := FormatProgress(75.5); got != "75.5%" {
		t.Errorf("FormatProgress(75.5) = %q, want %q", got, "75.5%")
	}
}

func TestHistoryManager(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	hm := NewHistoryManager(tempDir)

	// Create a test run
	run := &HistoricalRun{
		SubForumID:        "test-forum",
		TotalPages:        100,
		TotalTopics:       1000,
		UniqueTopicsFound: 950,
		TotalTime:         30.0,
		AveragePageRate:   3.33,
		AverageTopicRate:  33.3,
	}

	// Test saving a run
	if err := hm.SaveRun(run); err != nil {
		t.Errorf("Failed to save run: %v", err)
	}

	// Test retrieving recent runs
	runs, err := hm.GetRecentRuns(1)
	if err != nil {
		t.Errorf("Failed to get recent runs: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("Expected 1 run, got %d", len(runs))
	}

	// Test average rates calculation
	avgPageRate, avgTopicRate, err := hm.GetAverageRates(1)
	if err != nil {
		t.Errorf("Failed to get average rates: %v", err)
	}
	if avgPageRate != 3.33 {
		t.Errorf("Expected average page rate 3.33, got %.2f", avgPageRate)
	}
	if avgTopicRate != 33.3 {
		t.Errorf("Expected average topic rate 33.3, got %.2f", avgTopicRate)
	}

	// Test ETC estimation
	etc, err := hm.EstimateETC("new-forum", 100, 1000)
	if err != nil {
		t.Errorf("Failed to estimate ETC: %v", err)
	}
	if etc <= 0 {
		t.Errorf("Expected positive ETC, got %v", etc)
	}
}

func BenchmarkUpdateMetrics(b *testing.B) {
	tracker := NewTracker(10000, 100000)
	for i := 0; i < b.N; i++ {
		tracker.UpdateMetrics(i%10000, i%100000, i%100000)
	}
}

func BenchmarkGetETC(b *testing.B) {
	tracker := NewTracker(10000, 100000)
	tracker.UpdateMetrics(5000, 50000, 50000)
	for i := 0; i < b.N; i++ {
		_, _ = tracker.GetETC()
	}
}
