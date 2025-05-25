package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ProgressData holds the structure for the progress.json file
type ProgressData struct {
	OverallArchivalProgress float64 `json:"overall_archival_progress"`
	LastProcessedSubForum   string  `json:"last_processed_sub_forum"`
	LastProcessedTopic      string  `json:"last_processed_topic"`
	LastProcessedPage       string  `json:"last_processed_page"`
}

// Custom error types
var (
	ErrStorageNotFound      = errors.New("storage: item not found")
	ErrStorageInvalidFormat = errors.New("storage: invalid data format")
	ErrStoragePermission    = errors.New("storage: permission denied")
	ErrStorageFull          = errors.New("storage: storage is full") // For future use, hard to detect proactively
	ErrStorageIO            = errors.New("storage: I/O error")
)

// Logger setup
var (
	infoLog  *log.Logger
	warnLog  *log.Logger
	errorLog *log.Logger
)

func init() {
	// By default, log to os.Stdout for info/warn, os.Stderr for error.
	// Using Lshortfile can be performance-intensive in production.
	// Consider making flags and output configurable if this package were more general.
	infoLog = log.New(os.Stdout, "INFO: storage: ", log.Ldate|log.Ltime)
	warnLog = log.New(os.Stdout, "WARN: storage: ", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stderr, "ERROR: storage: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// SetLoggerOutput allows redirecting log output, useful for testing or custom handling.
func SetLoggerOutput(infoDest, warnDest, errorDest io.Writer) {
	if infoDest != nil {
		infoLog = log.New(infoDest, "INFO: storage: ", log.Ldate|log.Ltime)
	}
	if warnDest != nil {
		warnLog = log.New(warnDest, "WARN: storage: ", log.Ldate|log.Ltime)
	}
	if errorDest != nil {
		errorLog = log.New(errorDest, "ERROR: storage: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

// SubForumMetadata holds the structure for the sub-forum metadata index.json file
type SubForumMetadata struct {
	TotalTopics         int            `json:"total_topics"`
	PagesPerTopic       map[string]int `json:"pages_per_topic"` // Maps TopicID to its page count
	LastUpdateTimestamp string         `json:"last_update_timestamp"`
}

// InitializeStorage sets up the basic directory structure and progress file.
// It creates 'raw-html', 'structured-json', and 'metadata' directories under basePath.
// It also creates a 'progress.json' file in basePath if it doesn't exist.
func InitializeStorage(basePath string) error {
	dirs := []string{
		filepath.Join(basePath, "raw-html"),
		filepath.Join(basePath, "structured-json"),
		filepath.Join(basePath, "metadata"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	progressFilePath := filepath.Join(basePath, "progress.json")
	if _, err := os.Stat(progressFilePath); os.IsNotExist(err) {
		initialProgress := ProgressData{
			OverallArchivalProgress: 0.0,
			LastProcessedSubForum:   "",
			LastProcessedTopic:      "",
			LastProcessedPage:       "",
		}
		jsonData, err := json.MarshalIndent(initialProgress, "", "  ")
		if err != nil {
			return fmt.Errorf("%w: marshaling initial progress data: %s", ErrStorageInvalidFormat, err.Error())
		}
		if err := os.WriteFile(progressFilePath, jsonData, 0644); err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%w: writing initial progress.json %s: %s", ErrStoragePermission, progressFilePath, err.Error())
			}
			if strings.Contains(strings.ToLower(err.Error()), "no space left on device") {
				return fmt.Errorf("%w: writing initial progress.json %s: %s", ErrStorageFull, progressFilePath, err.Error())
			}
			return fmt.Errorf("%w: writing initial progress.json %s: %s", ErrStorageIO, progressFilePath, err.Error())
		}
	} else if err != nil {
		return fmt.Errorf("%w: stating progress.json %s: %s", ErrStorageIO, progressFilePath, err.Error())
	}

	return nil
}

// GetRawHTMLPath generates the full path for a raw HTML file.
func GetRawHTMLPath(basePath, subForumID, topicID, pageNumber string) string {
	return filepath.Join(basePath, "raw-html", "subforum-"+subForumID, "topic-"+topicID, "page-"+pageNumber+".html")
}

// GetStructuredJSONPath generates the full path for a structured JSON file.
func GetStructuredJSONPath(basePath, subForumID, topicID string) string {
	return filepath.Join(basePath, "structured-json", "subforum-"+subForumID, "topic-"+topicID+".json")
}

// GetSubForumMetadataIndexPath generates the full path for a sub-forum's metadata index file.
func GetSubForumMetadataIndexPath(basePath, subForumID string) string {
	return filepath.Join(basePath, "metadata", "subforum-"+subForumID, "index.json")
}

// WriteSubForumMetadata creates or updates a sub-forum's metadata index.json file.
// It ensures the directory for the metadata file is created.
func WriteSubForumMetadata(basePath, subForumID string, metadata SubForumMetadata) error {
	metadataFilePath := GetSubForumMetadataIndexPath(basePath, subForumID)

	// Ensure the directory exists
	metaDir := filepath.Dir(metadataFilePath)
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: marshaling subforum metadata: %s", ErrStorageInvalidFormat, err.Error())
	}

	err = os.WriteFile(metadataFilePath, jsonData, 0644)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("%w: writing metadata file %s: %s", ErrStoragePermission, metadataFilePath, err.Error())
		}
		if strings.Contains(strings.ToLower(err.Error()), "no space left on device") {
			return fmt.Errorf("%w: writing metadata file %s: %s", ErrStorageFull, metadataFilePath, err.Error())
		}
		return fmt.Errorf("%w: writing metadata file %s: %s", ErrStorageIO, metadataFilePath, err.Error())
	}
	return nil
}

// ReadProgressFile reads the global progress.json file.
func ReadProgressFile(basePath string) (ProgressData, error) {
	progressFilePath := filepath.Join(basePath, "progress.json")
	var data ProgressData

	fileContent, err := os.ReadFile(progressFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return data, fmt.Errorf("%w: progress.json at %s", ErrStorageNotFound, progressFilePath)
		}
		return data, fmt.Errorf("%w: reading progress.json %s: %s", ErrStorageIO, progressFilePath, err.Error())
	}

	err = json.Unmarshal(fileContent, &data)
	if err != nil {
		return data, fmt.Errorf("%w: unmarshaling progress.json: %s", ErrStorageInvalidFormat, err.Error())
	}

	// Basic validation
	if data.OverallArchivalProgress < 0 || data.OverallArchivalProgress > 100 {
		return data, fmt.Errorf("%w: OverallArchivalProgress out of range (0-100) in progress.json: %f", ErrStorageInvalidFormat, data.OverallArchivalProgress)
	}

	return data, nil
}

// WriteProgressFile writes data to the global progress.json file.
func WriteProgressFile(basePath string, data ProgressData) error {
	progressFilePath := filepath.Join(basePath, "progress.json")
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: marshaling progress data: %s", ErrStorageInvalidFormat, err.Error())
	}
	err = os.WriteFile(progressFilePath, jsonData, 0644)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("%w: writing progress.json %s: %s", ErrStoragePermission, progressFilePath, err.Error())
		}
		// Simplistic check for disk full - this is not reliable across systems/errors
		// A more robust solution would involve platform-specific checks or parsing specific error strings/codes
		if strings.Contains(strings.ToLower(err.Error()), "no space left on device") {
			return fmt.Errorf("%w: writing progress.json %s: %s", ErrStorageFull, progressFilePath, err.Error())
		}
		return fmt.Errorf("%w: writing progress.json %s: %s", ErrStorageIO, progressFilePath, err.Error())
	}
	return nil
}

// ValidateBaseStorageStructure checks if the essential directories and progress file exist.
func ValidateBaseStorageStructure(basePath string) error {
	expectedDirs := []string{
		filepath.Join(basePath, "raw-html"),
		filepath.Join(basePath, "structured-json"),
		filepath.Join(basePath, "metadata"),
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return err // Could wrap this error for more context
		} else if err != nil {
			return err
		}
	}

	progressFilePath := filepath.Join(basePath, "progress.json")
	if _, err := os.Stat(progressFilePath); os.IsNotExist(err) {
		return err // Could wrap this error for more context
	} else if err != nil {
		return err
	}

	return nil
}

// ParseRawHTMLPath extracts subForumID, topicID, and pageNumber from a raw HTML file path.
// It also infers and returns the basePath.
func ParseRawHTMLPath(filePath string) (basePath, subForumID, topicID, pageNumber string, err error) {
	// Expected structure: .../basePath/raw-html/subforum-SID/topic-TID/page-PN.html
	bn := filepath.Base(filePath)       // page-PN.html
	topicDir := filepath.Dir(filePath)  // .../basePath/raw-html/subforum-SID/topic-TID
	sfDir := filepath.Dir(topicDir)     // .../basePath/raw-html/subforum-SID
	rawHTMLDir := filepath.Dir(sfDir)   // .../basePath/raw-html
	basePath = filepath.Dir(rawHTMLDir) // .../basePath

	if filepath.Base(rawHTMLDir) != "raw-html" ||
		!strings.HasPrefix(filepath.Base(sfDir), "subforum-") ||
		!strings.HasPrefix(filepath.Base(topicDir), "topic-") ||
		!strings.HasPrefix(bn, "page-") || !strings.HasSuffix(bn, ".html") {
		return "", "", "", "", fmt.Errorf("invalid raw HTML path structure: %s", filePath)
	}

	subForumID = strings.TrimPrefix(filepath.Base(sfDir), "subforum-")
	topicID = strings.TrimPrefix(filepath.Base(topicDir), "topic-")
	pageNumber = strings.TrimSuffix(strings.TrimPrefix(bn, "page-"), ".html")

	if subForumID == "" || topicID == "" || pageNumber == "" {
		return "", "", "", "", fmt.Errorf("empty ID component in raw HTML path: %s", filePath)
	}

	return basePath, subForumID, topicID, pageNumber, nil
}

// ParseStructuredJSONPath extracts subForumID and topicID from a structured JSON file path.
// It also infers and returns the basePath.
func ParseStructuredJSONPath(filePath string) (basePath, subForumID, topicID string, err error) {
	// Expected structure: .../basePath/structured-json/subforum-SID/topic-TID.json
	bn := filepath.Base(filePath)              // topic-TID.json
	sfDir := filepath.Dir(filePath)            // .../basePath/structured-json/subforum-SID
	structuredJSONDir := filepath.Dir(sfDir)   // .../basePath/structured-json
	basePath = filepath.Dir(structuredJSONDir) // .../basePath

	if filepath.Base(structuredJSONDir) != "structured-json" ||
		!strings.HasPrefix(filepath.Base(sfDir), "subforum-") ||
		!strings.HasPrefix(bn, "topic-") || !strings.HasSuffix(bn, ".json") {
		return "", "", "", fmt.Errorf("invalid structured JSON path structure: %s", filePath)
	}

	subForumID = strings.TrimPrefix(filepath.Base(sfDir), "subforum-")
	topicID = strings.TrimSuffix(strings.TrimPrefix(bn, "topic-"), ".json")

	if subForumID == "" || topicID == "" {
		return "", "", "", fmt.Errorf("empty ID component in structured JSON path: %s", filePath)
	}

	return basePath, subForumID, topicID, nil
}

// ParseSubForumMetadataIndexPath extracts subForumID from a metadata index file path.
// It also infers and returns the basePath.
func ParseSubForumMetadataIndexPath(filePath string) (basePath, subForumID string, err error) {
	// Expected structure: .../basePath/metadata/subforum-SID/index.json
	//bn := filepath.Base(filePath) // index.json
	sfDir := filepath.Dir(filePath)      // .../basePath/metadata/subforum-SID
	metadataDir := filepath.Dir(sfDir)   // .../basePath/metadata
	basePath = filepath.Dir(metadataDir) // .../basePath

	if filepath.Base(metadataDir) != "metadata" ||
		!strings.HasPrefix(filepath.Base(sfDir), "subforum-") ||
		filepath.Base(filePath) != "index.json" {
		return "", "", fmt.Errorf("invalid metadata index path structure: %s", filePath)
	}

	subForumID = strings.TrimPrefix(filepath.Base(sfDir), "subforum-")

	if subForumID == "" {
		return "", "", fmt.Errorf("empty ID component in metadata index path: %s", filePath)
	}

	return basePath, subForumID, nil
}

// ReadSubForumMetadata reads and validates a sub-forum's metadata index.json file.
func ReadSubForumMetadata(basePath, subForumID string) (SubForumMetadata, error) {
	metadataFilePath := GetSubForumMetadataIndexPath(basePath, subForumID)
	var metadata SubForumMetadata

	fileContent, err := os.ReadFile(metadataFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return metadata, fmt.Errorf("%w: metadata file %s", ErrStorageNotFound, metadataFilePath)
		}
		return metadata, fmt.Errorf("%w: reading metadata file %s: %s", ErrStorageIO, metadataFilePath, err.Error())
	}

	err = json.Unmarshal(fileContent, &metadata)
	if err != nil {
		return metadata, fmt.Errorf("%w: unmarshaling metadata %s: %s", ErrStorageInvalidFormat, metadataFilePath, err.Error())
	}

	// Basic validation
	if metadata.TotalTopics < 0 {
		return metadata, fmt.Errorf("%w: TotalTopics invalid in %s: %d", ErrStorageInvalidFormat, metadataFilePath, metadata.TotalTopics)
	}
	// Potentially validate LastUpdateTimestamp format if needed, e.g. time.Parse

	return metadata, nil
}

// GetDirectorySize calculates the total size of all files within a directory (recursive).
func GetDirectorySize(dirPath string) (int64, error) {
	var totalSize int64
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err // Propagate errors from WalkDir itself
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				// This can happen if the file is removed between WalkDir finding it and d.Info() call
				// Or if there are permission issues. Depending on desired robustness, could log and continue.
				return fmt.Errorf("failed to get file info for %s: %w", path, err)
			}
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("error walking directory %s: %w", dirPath, err)
	}
	return totalSize, nil
}

// StorageStatus holds information about storage usage and warnings.
type StorageStatus struct {
	CurrentUsageBytes int64
	QuotaBytes        int64   // 0 or negative means no quota
	UsagePercentage   float64 // 0 if no quota
	IsWarning         bool
	Error             error
}

// CheckStorageQuota checks the storage usage against a quota and logs metrics.
// warningThresholdPercentage is e.g., 80.0 for 80%.
// quotaBytes <= 0 means no quota is set.
func CheckStorageQuota(basePath string, warningThresholdPercentage float64, quotaBytes int64) StorageStatus {
	status := StorageStatus{QuotaBytes: quotaBytes}

	currentUsage, err := GetDirectorySize(basePath)
	if err != nil {
		status.Error = fmt.Errorf("failed to get directory size for quota check: %w", err)
		errorLog.Printf("Error checking storage quota for %s: %v", basePath, status.Error)
		return status
	}
	status.CurrentUsageBytes = currentUsage

	if quotaBytes > 0 {
		status.UsagePercentage = (float64(currentUsage) / float64(quotaBytes)) * 100.0
		if status.UsagePercentage >= warningThresholdPercentage {
			status.IsWarning = true
			warnLog.Printf("Storage usage for %s is at %.2f%% (Used: %d, Quota: %d), exceeding threshold of %.2f%%.",
				basePath, status.UsagePercentage, currentUsage, quotaBytes, warningThresholdPercentage)
		} else {
			infoLog.Printf("Storage usage for %s is at %.2f%% (Used: %d, Quota: %d). Threshold: %.2f%%.",
				basePath, status.UsagePercentage, currentUsage, quotaBytes, warningThresholdPercentage)
		}
	} else {
		infoLog.Printf("Storage usage for %s is %d bytes. No quota set.", basePath, currentUsage)
	}
	return status
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: source file %s for copy", ErrStorageNotFound, src)
		}
		return fmt.Errorf("%w: reading source file %s for copy: %s", ErrStorageIO, src, err.Error())
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("%w: writing destination file %s for copy: %s", ErrStoragePermission, dst, err.Error())
		}
		if strings.Contains(strings.ToLower(err.Error()), "no space left on device") {
			return fmt.Errorf("%w: writing destination file %s for copy: %s", ErrStorageFull, dst, err.Error())
		}
		return fmt.Errorf("%w: writing destination file %s for copy: %s", ErrStorageIO, dst, err.Error())
	}
	return nil
}

// copyDirRecursive copies a directory recursively from src to dst.
// It creates dst if it doesn't exist.
func copyDirRecursive(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory %s: %w", src, err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source %s is not a directory", src)
	}

	// Create destination directory only after validating the source.
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirRecursive(srcPath, dstPath); err != nil {
				return err // Error already includes context
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err // Error already includes context
			}
		}
	}
	return nil
}

// BackupMetadataAndProgress creates a backup of progress.json and the entire metadata directory.
// backupBaseDir is the directory where timestamped backup folders will be created.
func BackupMetadataAndProgress(archiveBasePath, backupBaseDir string) (string, error) {
	infoLog.Printf("BackupMetadataAndProgress started. ArchiveBasePath: [%s], BackupBaseDir: [%s]", archiveBasePath, backupBaseDir)

	timestamp := time.Now().Format("20060102150405")
	backupDirName := fmt.Sprintf("project-waypoint-backup-%s", timestamp)
	fullBackupPath := filepath.Join(backupBaseDir, backupDirName)

	infoLog.Printf("Attempting to create backup target directory: [%s]", fullBackupPath)
	if err := os.MkdirAll(fullBackupPath, 0755); err != nil {
		errorLog.Printf("Failed to create main backup directory [%s]: %v", fullBackupPath, err)
		return "", fmt.Errorf("failed to create main backup directory %s: %w", fullBackupPath, err)
	}

	// 1. Backup progress.json
	progressFileSrc := filepath.Join(archiveBasePath, "progress.json")
	progressFileDst := filepath.Join(fullBackupPath, "progress.json")
	infoLog.Printf("Checking for progress file source: [%s]", progressFileSrc)
	if _, err := os.Stat(progressFileSrc); err == nil {
		infoLog.Printf("Progress file source [%s] exists. Attempting to copy to [%s]", progressFileSrc, progressFileDst)
		if errCopy := copyFile(progressFileSrc, progressFileDst); errCopy != nil {
			errorLog.Printf("Failed to backup progress.json from [%s] to [%s]: %v", progressFileSrc, progressFileDst, errCopy)
			return fullBackupPath, fmt.Errorf("failed to backup progress.json: %w", errCopy) // Return the actual copy error
		}
		infoLog.Printf("Successfully copied progress.json to [%s]", progressFileDst)
	} else if os.IsNotExist(err) {
		infoLog.Printf("Progress file source [%s] does not exist. Skipping copy.", progressFileSrc)
	} else {
		errorLog.Printf("Error stating progress file source [%s]: %v", progressFileSrc, err)
		return fullBackupPath, fmt.Errorf("%w: failed to stat source progress.json %s: %s", ErrStorageIO, progressFileSrc, err.Error())
	}

	// 2. Backup metadata directory
	metadataDirSrc := filepath.Join(archiveBasePath, "metadata")
	metadataDirDst := filepath.Join(fullBackupPath, "metadata")
	infoLog.Printf("Checking for metadata directory source: [%s]", metadataDirSrc)
	if statInfo, err := os.Stat(metadataDirSrc); err == nil {
		if !statInfo.IsDir() {
			warnLog.Printf("Source metadata path [%s] exists but is not a directory. Skipping copy.", metadataDirSrc)
		} else {
			infoLog.Printf("Metadata directory source [%s] exists and is a directory. Attempting to copy to [%s]", metadataDirSrc, metadataDirDst)
			if errCopy := copyDirRecursive(metadataDirSrc, metadataDirDst); errCopy != nil {
				errorLog.Printf("Failed to backup metadata directory from [%s] to [%s]: %v", metadataDirSrc, metadataDirDst, errCopy)
				return fullBackupPath, fmt.Errorf("failed to backup metadata directory: %w", errCopy) // Return actual copy error
			}
			infoLog.Printf("Successfully copied metadata directory to [%s]", metadataDirDst)
		}
	} else if os.IsNotExist(err) {
		infoLog.Printf("Metadata directory source [%s] does not exist. Skipping copy.", metadataDirSrc)
	} else {
		errorLog.Printf("Error stating metadata directory source [%s]: %v", metadataDirSrc, err)
		return fullBackupPath, fmt.Errorf("%w: failed to stat source metadata directory %s: %s", ErrStorageIO, metadataDirSrc, err.Error())
	}

	infoLog.Printf("BackupMetadataAndProgress finished successfully. Backup at: [%s]", fullBackupPath)
	return fullBackupPath, nil
}

// BackupInfo holds information about a single backup.
type BackupInfo struct {
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"` // Parsed from the directory name
	// Size      int64     `json:"size"` // Could be added by calculating size of backupDir
}

// ListBackups scans the backupBaseDir for valid backup directories and returns info about them.
func ListBackups(backupBaseDir string) ([]BackupInfo, error) {
	var backups []BackupInfo

	entries, err := os.ReadDir(backupBaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return backups, nil // No backup directory yet, not an error
		}
		return nil, fmt.Errorf("failed to read backup base directory %s: %w", backupBaseDir, err)
	}

	backupDirPrefix := "project-waypoint-backup-"
	timestampFormat := "20060102150405"

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), backupDirPrefix) {
			timestampStr := strings.TrimPrefix(entry.Name(), backupDirPrefix)
			timestamp, err := time.Parse(timestampFormat, timestampStr)
			if err == nil {
				backups = append(backups, BackupInfo{
					Path:      filepath.Join(backupBaseDir, entry.Name()),
					Timestamp: timestamp,
				})
			} else {
				// Log or handle directories that match prefix but have bad timestamp format
				warnLog.Printf("Found directory %s with backup prefix but invalid timestamp format: %v", entry.Name(), err)
			}
		}
	}

	// Sort backups by timestamp, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}
