package metrics

import (
	// "log" // Replaced by custom logger
	"sync"
	"time"

	"waypoint_archive_scripts/internal/indexer/logger" // Corrected
)

// MetricsTracker holds performance metrics for an indexing run.
// It uses a mutex to allow safe concurrent updates if necessary in the future,
// though current main.go usage is sequential.
// TODO: Expand with more granular metrics as needed from Story 1.5
// (e.g., http errors, parsing errors, time per stage)

type MetricsTracker struct {
	mutex              sync.Mutex
	StartTime          time.Time
	EndTime            time.Time
	PagesFetched       int
	TopicsFound        int
	TopicsAddedToStore int // Could be different from TopicsFound if de-duplication occurs across passes
	HTTPRequests       int
	SuccessfulRequests int
	FailedRequests     int
	CurrentPage        int // For ETC calculation
	TotalPages         int // For ETC calculation, set once pagination is known
}

// NewMetricsTracker creates and initializes a new MetricsTracker.
func NewMetricsTracker() *MetricsTracker {
	// Initial log message will be done by the main application after logger is initialized.
	return &MetricsTracker{
		StartTime: time.Now(),
	}
}

// SetTotalPages sets the total number of pages to be processed, used for ETC.
func (mt *MetricsTracker) SetTotalPages(totalPages int) {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.TotalPages = totalPages
	logger.Infof("[Metrics] Total pages for ETC calculation set to: %d", totalPages)
}

// IncrementPagesFetched increments the count of pages successfully fetched and processed.
// It also updates CurrentPage for ETC.
func (mt *MetricsTracker) IncrementPagesFetched() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.PagesFetched++
	mt.CurrentPage = mt.PagesFetched // Assuming one processed page = one step towards total
}

// IncrementHTTPRequests increments the count of total HTTP requests made.
func (mt *MetricsTracker) IncrementHTTPRequests() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.HTTPRequests++
}

// IncrementSuccessfulRequests increments the count of successful HTTP requests.
func (mt *MetricsTracker) IncrementSuccessfulRequests() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.SuccessfulRequests++
	// Also counts as a general HTTP request
	// Call IncrementHTTPRequests separately where the request is made.
}

// IncrementFailedRequests increments the count of failed HTTP requests.
func (mt *MetricsTracker) IncrementFailedRequests() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.FailedRequests++
	// Also counts as a general HTTP request
	// Call IncrementHTTPRequests separately where the request is made.
}

// AddTopicsFound increments the count of topics found (before de-duplication within a single pass's findings).
func (mt *MetricsTracker) AddTopicsFound(count int) {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.TopicsFound += count
}

// SetTopicsAddedToStore sets the final count of unique topics added to the store.
func (mt *MetricsTracker) SetTopicsAddedToStore(count int) {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()
	mt.TopicsAddedToStore = count
}

// LogETC calculates and logs the Estimated Time to Completion.
// This should be called periodically, e.g., after processing each page.
func (mt *MetricsTracker) LogETC() {
	mt.mutex.Lock()
	// Unlock without defer as we are only reading and don't want to hold lock during log print
	// which might do I/O. Copy values needed.
	currentPage := mt.CurrentPage
	totalPages := mt.TotalPages
	startTime := mt.StartTime
	mt.mutex.Unlock()

	if totalPages == 0 || currentPage == 0 || currentPage > totalPages {
		// Not enough data or already past the end
		return
	}

	elapsedTime := time.Since(startTime)
	progress := float64(currentPage) / float64(totalPages)
	if progress == 0 { // Avoid division by zero if called too early
		return
	}
	estimatedTotalTime := elapsedTime.Seconds() / progress
	remainingTimeSeconds := estimatedTotalTime - elapsedTime.Seconds()

	if remainingTimeSeconds < 0 {
		remainingTimeSeconds = 0 // Can happen if current page is very close to total pages
	}

	remainingDuration := time.Duration(remainingTimeSeconds) * time.Second

	logger.Infof("[Metrics] ETC: Processed %d/%d pages (%.2f%%). Approx. %s remaining.",
		currentPage, totalPages, progress*100, remainingDuration.Round(time.Second))
}

// FinalizeAndLogMetrics sets the end time and logs all collected metrics.
func (mt *MetricsTracker) FinalizeAndLogMetrics() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	mt.EndTime = time.Now()
	duration := mt.EndTime.Sub(mt.StartTime)

	logger.Printf("--- Performance Metrics --- ") // Using Printf for section headers
	logger.Infof("[Metrics] Indexing Run Start Time: %s", mt.StartTime.Format(time.RFC3339))
	logger.Infof("[Metrics] Indexing Run End Time:   %s", mt.EndTime.Format(time.RFC3339))
	logger.Infof("[Metrics] Total Duration:           %s", duration.Round(time.Second))
	logger.Infof("[Metrics] Total Pages Processed:    %d", mt.PagesFetched)
	logger.Infof("[Metrics] Total HTTP Requests:      %d (Successful: %d, Failed: %d)", mt.HTTPRequests, mt.SuccessfulRequests, mt.FailedRequests)
	logger.Infof("[Metrics] Total Topics Found (raw): %d", mt.TopicsFound) // Before final de-duplication
	logger.Infof("[Metrics] Unique Topics Saved:      %d", mt.TopicsAddedToStore)

	if mt.PagesFetched > 0 && duration.Seconds() > 0 {
		avgTimePerPage := duration.Seconds() / float64(mt.PagesFetched)
		logger.Infof("[Metrics] Avg. Time Per Page:     %.2f seconds", avgTimePerPage)
	}
	if mt.TopicsAddedToStore > 0 && duration.Seconds() > 0 {
		topicsPerSecond := float64(mt.TopicsAddedToStore) / duration.Seconds()
		logger.Infof("[Metrics] Avg. Topics Saved/Sec:  %.2f", topicsPerSecond)
	}
	logger.Printf("---------------------------") // Using Printf for section headers
}
