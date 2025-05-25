package metrics

import (
	"fmt"
	"time"
)

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes()) // Convert to int to round down
		return fmt.Sprintf("%d minutes", minutes)
	}
	hours := d.Hours()
	if hours < 24 {
		return fmt.Sprintf("%.1f hours", hours)
	}
	days := hours / 24
	return fmt.Sprintf("%.1f days", days)
}

// FormatRate formats a processing rate in a human-readable way
func FormatRate(rate float64, unit string) string {
	if rate < 1 {
		return fmt.Sprintf("%.2f %s/minute", rate, unit)
	}
	return fmt.Sprintf("%.0f %s/minute", rate, unit)
}

// FormatProgress formats a progress percentage in a human-readable way
func FormatProgress(progress float64) string {
	return fmt.Sprintf("%.1f%%", progress)
}

// FormatMetrics formats all current metrics in a human-readable way
func FormatMetrics(m *ProcessingMetrics) string {
	etc, err := m.GetETC()
	etcStr := "calculating..."
	if err == nil {
		etcStr = FormatDuration(etc)
	}

	pageProgress, topicProgress := m.GetProgress()
	currentPageRate, currentTopicRate := m.GetCurrentRates()
	avgPageRate, avgTopicRate := m.GetAverageRates()

	return fmt.Sprintf(
		"Progress: %s (pages) / %s (topics)\n"+
			"Current Rate: %s / %s\n"+
			"Average Rate: %s / %s\n"+
			"Unique Topics Found: %d\n"+
			"Elapsed Time: %s\n"+
			"ETC: %s",
		FormatProgress(pageProgress),
		FormatProgress(topicProgress),
		FormatRate(currentPageRate, "pages"),
		FormatRate(currentTopicRate, "topics"),
		FormatRate(avgPageRate, "pages"),
		FormatRate(avgTopicRate, "topics"),
		m.GetUniqueTopicsCount(),
		FormatDuration(m.GetElapsedTime()),
		etcStr,
	)
}
