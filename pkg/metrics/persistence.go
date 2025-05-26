package metrics

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	// DefaultPerformanceLogPath is the default location for the performance log file
	DefaultPerformanceLogPath = "logs/performance_log.csv"
)

// PerformanceLogger handles saving and loading historical performance metrics
type PerformanceLogger struct {
	logPath string
}

// NewPerformanceLogger creates a new PerformanceLogger with the specified log path
func NewPerformanceLogger(logPath string) *PerformanceLogger {
	if logPath == "" {
		logPath = DefaultPerformanceLogPath
	}
	return &PerformanceLogger{logPath: logPath}
}

// AppendMetrics saves a single batch's metrics to the performance log file
func (l *PerformanceLogger) AppendMetrics(metrics HistoricalMetrics) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(l.logPath), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(l.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if file is empty
	if fileInfo, err := file.Stat(); err == nil && fileInfo.Size() == 0 {
		header := []string{
			"TimestampUTC", "BatchID", "DurationSeconds", "PagesArchived",
			"TopicsArchived", "BytesArchived", "AvgPagesPerMin",
			"AvgTopicsPerHour", "AvgMBPerMin",
		}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	// Convert metrics to CSV row
	row := []string{
		metrics.TimestampUTC.Format(time.RFC3339),
		metrics.BatchID,
		fmt.Sprintf("%.2f", metrics.DurationSeconds),
		strconv.FormatInt(metrics.PagesArchived, 10),
		strconv.FormatInt(metrics.TopicsArchived, 10),
		strconv.FormatInt(metrics.BytesArchived, 10),
		fmt.Sprintf("%.2f", metrics.AvgPagesPerMin),
		fmt.Sprintf("%.2f", metrics.AvgTopicsPerHour),
		fmt.Sprintf("%.2f", metrics.AvgMBPerMin),
	}

	if err := writer.Write(row); err != nil {
		return fmt.Errorf("failed to write metrics row: %w", err)
	}

	return nil
}

// LoadHistoricalMetrics reads all historical metrics from the log file
func (l *PerformanceLogger) LoadHistoricalMetrics() ([]HistoricalMetrics, error) {
	file, err := os.OpenFile(l.logPath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var metrics []HistoricalMetrics
	for {
		record, err := reader.Read()
		if err != nil {
			break // EOF or error
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		// Parse numeric values
		duration, _ := strconv.ParseFloat(record[2], 64)
		pages, _ := strconv.ParseInt(record[3], 10, 64)
		topics, _ := strconv.ParseInt(record[4], 10, 64)
		bytes, _ := strconv.ParseInt(record[5], 10, 64)
		avgPages, _ := strconv.ParseFloat(record[6], 64)
		avgTopics, _ := strconv.ParseFloat(record[7], 64)
		avgMB, _ := strconv.ParseFloat(record[8], 64)

		metrics = append(metrics, HistoricalMetrics{
			TimestampUTC:     timestamp,
			BatchID:          record[1],
			DurationSeconds:  duration,
			PagesArchived:    pages,
			TopicsArchived:   topics,
			BytesArchived:    bytes,
			AvgPagesPerMin:   avgPages,
			AvgTopicsPerHour: avgTopics,
			AvgMBPerMin:      avgMB,
		})
	}

	return metrics, nil
}

// CalculateAverageRates computes average rates from historical metrics
func (l *PerformanceLogger) CalculateAverageRates() (float64, float64, float64, error) {
	metrics, err := l.LoadHistoricalMetrics()
	if err != nil {
		return 0, 0, 0, err
	}

	if len(metrics) == 0 {
		return 0, 0, 0, nil
	}

	var totalPagesPerMin, totalTopicsPerHour, totalMBPerMin float64
	for _, m := range metrics {
		totalPagesPerMin += m.AvgPagesPerMin
		totalTopicsPerHour += m.AvgTopicsPerHour
		totalMBPerMin += m.AvgMBPerMin
	}

	count := float64(len(metrics))
	return totalPagesPerMin / count,
		totalTopicsPerHour / count,
		totalMBPerMin / count,
		nil
}
