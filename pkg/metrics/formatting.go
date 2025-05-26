package metrics

import (
	"fmt"
	"time"
)

// FormatDuration formats a time.Duration into a human-readable string
// e.g., "3 days 4 hours 15 minutes"
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "Calculating..."
	}

	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute

	if days > 0 {
		return fmt.Sprintf("%d days %d hours %d minutes", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	} else {
		return fmt.Sprintf("%d minutes", minutes)
	}
}

// FormatBytes formats a byte count into a human-readable string
// e.g., "1.5 MB", "2.3 GB"
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

// FormatProgress formats progress information into a human-readable string
// e.g., "Processed: 1500 of 11000 topics"
func FormatProgress(current, total int64) string {
	return fmt.Sprintf("Processed: %d of %d", current, total)
}

// FormatRates formats current processing rates into a human-readable string
// e.g., "Rate: 30 pages/min, 2 topics/hour, 8.14 MB/min"
func FormatRates(pagesPerMin, topicsPerHour, mbPerMin float64) string {
	return fmt.Sprintf("Rate: %.1f pages/min, %.1f topics/hour, %.2f MB/min",
		pagesPerMin, topicsPerHour, mbPerMin)
}

// FormatETC formats an ETC into a human-readable string
// e.g., "ETC: ~3 days 4 hours 15 minutes"
func FormatETC(etc time.Duration) string {
	if etc == 0 {
		return "ETC: Calculating..."
	}
	return fmt.Sprintf("ETC: ~%s", FormatDuration(etc))
}
