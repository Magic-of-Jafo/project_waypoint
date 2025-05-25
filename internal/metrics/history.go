package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// HistoricalRun represents the performance metrics from a completed sub-forum indexing run
type HistoricalRun struct {
	SubForumID        string    `json:"sub_forum_id"`
	Timestamp         time.Time `json:"timestamp"`
	TotalPages        int       `json:"total_pages"`
	TotalTopics       int       `json:"total_topics"`
	UniqueTopicsFound int       `json:"unique_topics_found"`
	TotalTime         float64   `json:"total_time_minutes"`
	AveragePageRate   float64   `json:"average_page_rate"`
	AverageTopicRate  float64   `json:"average_topic_rate"`
}

// HistoryManager manages the storage and retrieval of historical performance data
type HistoryManager struct {
	historyFilePath string
}

// NewHistoryManager creates a new history manager with the specified storage path
func NewHistoryManager(basePath string) *HistoryManager {
	historyDir := filepath.Join(basePath, "metrics")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		// Log error but continue - we'll handle file operations gracefully
		fmt.Printf("Warning: Failed to create metrics directory: %v\n", err)
	}
	return &HistoryManager{
		historyFilePath: filepath.Join(historyDir, "performance_history.jsonl"),
	}
}

// SaveRun saves the metrics from a completed run to the history file
func (hm *HistoryManager) SaveRun(run *HistoricalRun) error {
	// Create the run entry
	run.Timestamp = time.Now()

	// Convert to JSON
	jsonData, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal run data: %w", err)
	}

	// Append to file
	f, err := os.OpenFile(hm.historyFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write to history file: %w", err)
	}

	return nil
}

// GetRecentRuns retrieves the most recent performance data
func (hm *HistoryManager) GetRecentRuns(limit int) ([]HistoricalRun, error) {
	// Read the entire file
	data, err := os.ReadFile(hm.historyFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No history yet
		}
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	// Parse JSONL format
	var runs []HistoricalRun
	lines := bytes.Split(data, []byte{'\n'})
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var run HistoricalRun
		if err := json.Unmarshal(line, &run); err != nil {
			return nil, fmt.Errorf("failed to parse history entry: %w", err)
		}
		runs = append(runs, run)
	}

	// Sort by timestamp (most recent first)
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Timestamp.After(runs[j].Timestamp)
	})

	// Return only the requested number of runs
	if limit > 0 && limit < len(runs) {
		runs = runs[:limit]
	}

	return runs, nil
}

// GetAverageRates calculates the average processing rates from recent runs
func (hm *HistoryManager) GetAverageRates(limit int) (avgPageRate, avgTopicRate float64, err error) {
	runs, err := hm.GetRecentRuns(limit)
	if err != nil {
		return 0, 0, err
	}
	if len(runs) == 0 {
		return 0, 0, fmt.Errorf("no historical data available")
	}

	var totalPageRate, totalTopicRate float64
	for _, run := range runs {
		totalPageRate += run.AveragePageRate
		totalTopicRate += run.AverageTopicRate
	}

	return totalPageRate / float64(len(runs)), totalTopicRate / float64(len(runs)), nil
}

// EstimateETC calculates an initial ETC for a new sub-forum based on historical data
func (hm *HistoryManager) EstimateETC(subForumID string, expectedPages, expectedTopics int) (time.Duration, error) {
	avgPageRate, _, err := hm.GetAverageRates(10) // Use last 10 runs
	if err != nil {
		return 0, fmt.Errorf("failed to get historical rates: %w", err)
	}

	if avgPageRate <= 0 {
		return 0, fmt.Errorf("invalid average page rate from history")
	}

	// Calculate ETC based on pages and average rate
	etcMinutes := float64(expectedPages) / avgPageRate
	return time.Duration(etcMinutes * float64(time.Minute)), nil
}
