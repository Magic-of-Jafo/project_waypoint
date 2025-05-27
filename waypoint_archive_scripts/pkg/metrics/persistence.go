package metrics

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// Constants for performance metrics
const (
	DefaultPerformanceLogPath = "logs/performance_log.csv"
)

// Global metrics logger instance and mutex for thread-safe operations
var (
	perfLogger *PerformanceLogger
	once       sync.Once
	loggerMux  sync.Mutex
)

// PerformanceLogger handles saving and loading historical performance metrics
type PerformanceLogger struct {
	LogFilePath string
	buffer      []PerformanceMetric
	mux         sync.Mutex
}

// NewPerformanceLogger creates a new PerformanceLogger instance.
// DEPRECATED: Use InitPerformanceLogger for global instance.
func NewPerformanceLogger(logPath string) *PerformanceLogger {
	log.Printf("[WARNING] METRICS: NewPerformanceLogger is deprecated. Use InitPerformanceLogger for global instance.")
	return &PerformanceLogger{LogFilePath: logPath}
}

// InitPerformanceLogger initializes the global performance logger.
func InitPerformanceLogger(logFilePath string) {
	once.Do(func() {
		perfLogger = &PerformanceLogger{
			LogFilePath: logFilePath,
			buffer:      make([]PerformanceMetric, 0, 100),
		}
		// Ensure directory exists
		dir := filepath.Dir(logFilePath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Printf("[ERROR] METRICS: Failed to create directory for performance log %s: %v. Metrics might not be saved.", dir, err)
			}
		}
		log.Printf("[INFO] METRICS: Performance logger initialized. Log file: %s", logFilePath)
	})
}

// AppendMetric appends a single performance metric to the buffer.
func (pl *PerformanceLogger) AppendMetric(metric PerformanceMetric) {
	if pl == nil {
		log.Println("[ERROR] METRICS: AppendMetric called on nil PerformanceLogger.")
		return
	}
	pl.mux.Lock()
	defer pl.mux.Unlock()
	pl.buffer = append(pl.buffer, metric)
	log.Printf("[DEBUG] METRICS: Appended metric for %s. Buffer size: %d", metric.ResourceID, len(pl.buffer))
}

// SaveMetrics saves all buffered metrics to the CSV log file.
func (pl *PerformanceLogger) SaveMetrics() error {
	if pl == nil {
		return fmt.Errorf("SaveMetrics called on nil PerformanceLogger")
	}
	pl.mux.Lock()
	defer pl.mux.Unlock()

	if len(pl.buffer) == 0 {
		log.Println("[INFO] METRICS: No metrics in buffer to save.")
		return nil
	}

	dir := filepath.Dir(pl.LogFilePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for performance log %s: %w", dir, err)
		}
	}

	fileExists := true
	if _, err := os.Stat(pl.LogFilePath); os.IsNotExist(err) {
		fileExists = false
	}

	file, err := os.OpenFile(pl.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open performance log file %s: %w", pl.LogFilePath, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if !fileExists {
		header := []string{"Timestamp", "ResourceType", "ResourceID", "Action", "Size", "DurationMS", "RateMBps", "Notes"}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write CSV header to performance log: %w", err)
		}
	}

	for _, metric := range pl.buffer {
		record := []string{
			metric.Timestamp.Format(time.RFC3339Nano),
			string(metric.ResourceType),
			metric.ResourceID,
			string(metric.Action),
			strconv.FormatInt(metric.Size, 10),
			strconv.FormatInt(metric.Duration.Milliseconds(), 10),
			fmt.Sprintf("%.2f", metric.RateMBps),
			metric.Notes,
		}
		if err := writer.Write(record); err != nil {
			log.Printf("[ERROR] METRICS: Failed to write metric record to CSV for %s: %v. Skipping record.", metric.ResourceID, err)
		}
	}

	log.Printf("[INFO] METRICS: Successfully saved %d metrics to %s.", len(pl.buffer), pl.LogFilePath)
	pl.buffer = pl.buffer[:0]
	return nil
}

// LoadMetrics loads all metrics from the CSV log file.
func (pl *PerformanceLogger) LoadMetrics() ([]PerformanceMetric, error) {
	if pl == nil {
		return nil, fmt.Errorf("LoadMetrics called on nil PerformanceLogger")
	}
	pl.mux.Lock()
	defer pl.mux.Unlock()

	file, err := os.Open(pl.LogFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[INFO] METRICS: Performance log file %s does not exist. No metrics loaded.", pl.LogFilePath)
			return []PerformanceMetric{}, nil
		}
		return nil, fmt.Errorf("failed to open performance log file %s for reading: %w", pl.LogFilePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV records from performance log %s: %w", pl.LogFilePath, err)
	}

	var metrics []PerformanceMetric
	if len(records) <= 1 {
		return metrics, nil
	}

	for i, record := range records[1:] {
		if len(record) != 8 {
			log.Printf("[WARNING] METRICS: Skipping malformed record at line %d in %s: expected 8 fields, got %d", i+2, pl.LogFilePath, len(record))
			continue
		}
		timestamp, _ := time.Parse(time.RFC3339Nano, record[0])
		size, _ := strconv.ParseInt(record[4], 10, 64)
		durationMs, _ := strconv.ParseInt(record[5], 10, 64)
		rate, _ := strconv.ParseFloat(record[6], 64)

		metrics = append(metrics, PerformanceMetric{
			Timestamp:    timestamp,
			ResourceType: MetricResourceType(record[1]),
			ResourceID:   record[2],
			Action:       MetricAction(record[3]),
			Size:         size,
			Duration:     time.Duration(durationMs) * time.Millisecond,
			RateMBps:     rate,
			Notes:        record[7],
		})
	}
	return metrics, nil
}

// AppendMetrics saves a single batch's metrics to the performance log file
func (l *PerformanceLogger) AppendMetrics(metrics HistoricalMetrics) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(l.LogFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(l.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	file, err := os.OpenFile(l.LogFilePath, os.O_RDONLY|os.O_CREATE, 0644)
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

// RecordTopicPageArchived is a placeholder to record metrics for an archived topic page.
func RecordTopicPageArchived(topicID string, pageNum int, pageSizeBytes int64, duration time.Duration, logPath string) {
	// This function is a placeholder. The actual metric recording would use the
	// global perfLogger if initialized, or handle metrics in a simpler way for placeholders.
	// For consistency with other placeholders, we'll just log.
	loggerMux.Lock() // Use existing mutex from the package
	defer loggerMux.Unlock()

	// Attempt to use global logger if initialized by other calls, but don't make it a fatal error if not.
	if perfLogger != nil && perfLogger.LogFilePath != "" { // Check if perfLogger seems usable
		// Construct a metric similar to how perfLogger.AppendMetric would
		metric := PerformanceMetric{
			Timestamp:    time.Now(),
			ResourceType: ResourceTypeTopicPage,
			ResourceID:   fmt.Sprintf("%s_p%d", topicID, pageNum),
			Action:       ActionArchived,
			Size:         pageSizeBytes,
			Duration:     duration,
		}
		if duration > 0 {
			rate := (float64(pageSizeBytes) / (1024 * 1024)) / duration.Seconds()
			metric.RateMBps = rate
		}
		// perfLogger.AppendMetric(metric) // This would be the call to the actual logger
		log.Printf("[INFO] METRICS_PLACEHOLDER: Would append metric via perfLogger for Topic %s, Page %d. Size: %d, Duration: %s. Path: %s", topicID, pageNum, pageSizeBytes, duration, logPath)
	} else {
		// Fallback log if perfLogger isn't ready or configured for this placeholder context
		log.Printf("[INFO] METRICS_PLACEHOLDER: Recording (simple log) metric for Topic %s, Page %d. Size: %d, Duration: %s. Configured LogPath: %s", topicID, pageNum, pageSizeBytes, duration, logPath)
	}
}

// SavePerformanceLog is a placeholder to save all buffered performance metrics.
func SavePerformanceLog(logPath string) error {
	// This function is a placeholder. The actual save would use the global perfLogger.
	loggerMux.Lock()
	defer loggerMux.Unlock()

	if perfLogger != nil && perfLogger.LogFilePath == logPath && perfLogger.LogFilePath != "" {
		// log.Printf("[INFO] METRICS_PLACEHOLDER: Would call perfLogger.SaveMetrics() for path %s", logPath)
		// err := perfLogger.SaveMetrics() // This would be the call
		// if err != nil {
		// 	log.Printf("[ERROR] METRICS_PLACEHOLDER: perfLogger.SaveMetrics() failed: %v", err)
		// 	return err
		// }
		// log.Printf("[INFO] METRICS_PLACEHOLDER: perfLogger.SaveMetrics() successful for %s.", logPath)
		log.Printf("[INFO] METRICS_PLACEHOLDER: Simulating perfLogger.SaveMetrics() for path %s. (No actual save in this simplified placeholder version of SavePerformanceLog).", logPath)
		return nil
	} else {
		log.Printf("[INFO] METRICS_PLACEHOLDER: SavePerformanceLog called for path %s. Global perfLogger not configured for this path or not ready. No metrics saved by this placeholder.", logPath)
		return nil // Placeholder returns nil error
	}
}

// DisplayCurrentETC_Placeholder simulates displaying current ETC and processing rates.
func DisplayCurrentETC_Placeholder() {
	// This is a simple placeholder and does not access shared metrics state directly.
	// A real implementation would use perfLogger or a BatchMetrics instance.
	log.Printf("[INFO] METRICS_PLACEHOLDER: Would display current ETC and processing rates.")
}

// AppendDetailMetric provides a global way to append a detailed performance metric.
func AppendDetailMetric(metric PerformanceMetric) {
	if perfLogger == nil {
		log.Println("[ERROR] METRICS: Global performance logger not initialized. Cannot append metric.")
		return
	}
	perfLogger.AppendMetric(metric)
}

// SaveDetailMetricsLog provides a global way to save all buffered detailed metrics.
func SaveDetailMetricsLog() error {
	if perfLogger == nil {
		log.Println("[ERROR] METRICS: Global performance logger not initialized. Cannot save metrics.")
		return fmt.Errorf("global performance logger not initialized")
	}
	return perfLogger.SaveMetrics()
}
