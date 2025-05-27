package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration.
// For now, it's basic. It can be expanded to load from a file or env vars.
type Config struct {
	TopicIndexDir         string        `json:"topicIndexDir"`
	SubForumListFile      string        `json:"subForumListFile"`
	TopicIndexFilePattern string        `json:"topicIndexFilePattern"`
	PolitenessDelay       time.Duration `json:"politenessDelay"`
	UserAgent             string        `json:"userAgent"`
	ArchiveRootDir        string        `json:"archiveRootDir"`     // Added for archiver, used by storer (Story 2.4)
	StateFilePath         string        `json:"stateFilePath"`      // Added for Story 2.6
	SaveStateInterval     time.Duration `json:"saveStateInterval"`  // Added for Story 2.6, re-adding
	PerformanceLogPath    string        `json:"performanceLogPath"` // JSON tag updated for consistency
	ConfigFilePath        string        `json:"-"`                  // Path to config file, not part of JSON/env/cli itself usually

	// New fields for Story 2.8
	LogLevel             string        `json:"logLevel"`
	LogFilePath          string        `json:"logFilePath"`          // Optional: if empty, log to stdout/stderr
	JITRefreshPages      int           `json:"jitRefreshPages"`      // Number of initial pages of a sub-forum to re-scan
	JITRefreshInterval   time.Duration `json:"jitRefreshInterval"`   // How often to consider a JIT refresh for a sub-forum
	ForumBaseURL         string        `json:"forumBaseURL"`         // Base URL of the forum, e.g., http://forum.example.com/
	ArchiveOutputRootDir string        `json:"archiveOutputRootDir"` // Corrected JSON tag for consistency

	// TestConfiguration specific fields
	TestSubForumIDs       []string `json:"TestSubForumIDs,omitempty"`       // Match JSON key
	TestArchiveOutputRoot string   `json:"TestArchiveOutputRoot,omitempty"` // Match JSON key
}

// DefaultConfig returns a new Config with default values.
// These would typically be paths within a data directory.
func DefaultConfig() *Config {
	return &Config{
		TopicIndexDir:         "data/topic_indices",
		SubForumListFile:      "data/subforum_list.csv",
		TopicIndexFilePattern: "topic_index_forum_%s.csv", // Default was CSV, but we use JSON now, pattern will be from config.json ideally
		PolitenessDelay:       3 * time.Second,            // Default politeness delay
		UserAgent:             "WaypointArchiveAgent/1.0", // Default User-Agent
		ArchiveRootDir:        "archive_output",           // Default archive root
		StateFilePath:         "archive_progress.json",    // Default state file path (Story 2.6)
		SaveStateInterval:     5 * time.Minute,            // Default save state interval (Story 2.6), re-adding
		PerformanceLogPath:    "logs/performance_log.csv",
		LogLevel:              "INFO",                  // Default log level
		LogFilePath:           "",                      // Default: no log file, use stdout/stderr
		JITRefreshPages:       1,                       // Default: rescan 1 page for JIT refresh
		JITRefreshInterval:    24 * time.Hour,          // Default: consider JIT refresh once a day per sub-forum
		ConfigFilePath:        configFile,              // Default config file path
		ForumBaseURL:          "http://localhost:8080", // Placeholder, replace with actual default or leave empty
		ArchiveOutputRootDir:  "archive_output",        // This was `archive_output_root_dir` in JSON. Ensuring consistency.
		TestArchiveOutputRoot: "./test_archive_output", // Default for test runs
	}
}

const configFile = "config.json"

// LoadConfig loads the application configuration.
// It accepts command-line arguments (typically os.Args[1:]) for parsing.
// Order of precedence (lowest to highest):
// 1. Hardcoded default values.
// 2. Values from Environment Variables.
// 3. Values from a configuration file (if specified and exists).
// 4. Values from command-line flags.
func LoadConfig(arguments []string) (*Config, error) {
	// Start with hardcoded defaults
	cfg := DefaultConfig()

	// 2. Load from Environment Variables
	loadFromEnv(cfg)

	// Prepare to parse command-line flags to see if a custom config path is specified early.
	// We only look for -configFile / -c here. Other flags are parsed later.
	// This allows CLI to override the default config file path.
	prelimFlags := flag.NewFlagSet("prelim-config", flag.ContinueOnError)
	prelimFlags.String("configFile", "", "Path to JSON configuration file.")    // Value retrieved by lookup
	prelimFlags.String("c", "", "Path to JSON configuration file (shorthand).") // Shorthand, value retrieved by lookup

	// Parse preliminary flags. We don't care about errors here, just want the value if provided.
	// We also don't want it to exit on -help, so use a throwaway writer.
	prelimFlags.SetOutput(io.Discard)
	_ = prelimFlags.Parse(arguments)

	// Determine effective config file path
	effectiveConfigFile := cfg.ConfigFilePath // Default
	if val := prelimFlags.Lookup("configFile"); val != nil && val.Value.String() != "" {
		effectiveConfigFile = val.Value.String()
	} else if val := prelimFlags.Lookup("c"); val != nil && val.Value.String() != "" {
		// Check shorthand only if full flag not used
		effectiveConfigFile = val.Value.String()
	}

	// 3. Attempt to load from the configuration file
	if effectiveConfigFile != "" {
		if _, err := os.Stat(effectiveConfigFile); err == nil {
			data, err := os.ReadFile(effectiveConfigFile)
			if err != nil {
				log.Printf("[WARNING] Error reading config file %s: %v. Using defaults/CLI flags.", effectiveConfigFile, err)
			} else {
				err = json.Unmarshal(data, cfg)
				if err != nil {
					log.Printf("[WARNING] Error unmarshalling config file %s: %v. Using defaults/CLI flags.", effectiveConfigFile, err)
				} else {
					log.Printf("[INFO] Loaded configuration from %s", effectiveConfigFile)
					// Re-apply env vars over config file, in case some fields are not in JSON but set by ENV
					// And CLI will override both later.
					loadFromEnv(cfg)
				}
			}
		} else if !os.IsNotExist(err) {
			log.Printf("[WARNING] Error checking for config file %s: %v. Using defaults/CLI flags.", effectiveConfigFile, err)
		}
	}

	// 4. Define and parse all command-line flags using a local FlagSet
	// THIS SECTION IS TEMPORARILY COMMENTED OUT FOR DEBUGGING -- NOW RE-ENABLING

	configFlags := flag.NewFlagSet("archiver-config", flag.ContinueOnError)

	// Add configFile flag again to this set so it appears in help messages.
	// The value from prelimFlags is used for loading, this is for documentation and potential override if not already loaded.
	// However, since we load config file *before* this final flag parsing, this flag primarily serves for -help display.
	// To make it an actual override here would mean re-loading the config file, which is complex.
	// Simpler: configFile path is determined *before* full flag parsing.
	configFlags.String("configFile", effectiveConfigFile, "Path to JSON configuration file.")
	configFlags.String("c", effectiveConfigFile, "Path to JSON configuration file (shorthand).")

	cliTopicIndexDir := configFlags.String("topicIndexDir", cfg.TopicIndexDir, "Directory for topic index CSVs")
	cliSubForumListFile := configFlags.String("subForumListFile", cfg.SubForumListFile, "Path to subforum list CSV")
	cliTopicIndexFilePattern := configFlags.String("topicIndexFilePattern", cfg.TopicIndexFilePattern, "Pattern for topic index filenames, use %s for SubForumID")
	cliPolitenessDelay := configFlags.String("politenessDelay", cfg.PolitenessDelay.String(), "Politeness delay (e.g., '3s', '500ms')")
	cliUserAgent := configFlags.String("userAgent", cfg.UserAgent, "Custom User-Agent string")
	cliArchiveRootDir := configFlags.String("archiveRootDir", cfg.ArchiveRootDir, "Root directory for storing archived files")
	cliStateFilePath := configFlags.String("stateFilePath", cfg.StateFilePath, "Path to the archive progress state file")
	cliSaveStateInterval := configFlags.String("saveStateInterval", cfg.SaveStateInterval.String(), "Interval for saving state (e.g., '5m', '30s')")
	cliPerformanceLogPath := configFlags.String("performanceLogPath", cfg.PerformanceLogPath, "Path to the performance log file")
	cliLogLevel := configFlags.String("logLevel", cfg.LogLevel, "Logging level (DEBUG, INFO, WARN, ERROR)")
	cliLogFilePath := configFlags.String("logFilePath", cfg.LogFilePath, "Path to log file (optional, logs to stdout if empty)")
	cliJITRefreshPages := configFlags.Int("jitRefreshPages", cfg.JITRefreshPages, "Number of initial sub-forum pages to re-scan for JIT topic refresh")
	cliJITRefreshInterval := configFlags.String("jitRefreshInterval", cfg.JITRefreshInterval.String(), "How often to consider JIT refresh (e.g., '24h', '1h30m')")
	cliForumBaseURL := configFlags.String("forumBaseURL", cfg.ForumBaseURL, "Base URL of the target forum (e.g., http://forum.example.com)")
	cliArchiveOutputRootDir := configFlags.String("archiveOutputRootDir", cfg.ArchiveOutputRootDir, "Root directory for storing archived files")

	err := configFlags.Parse(arguments)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return cfg, err
		}
		// CRITICAL DEBUG: Print error directly to stderr
		fmt.Fprintf(os.Stderr, "[DEBUG_STDERR] config.LoadConfig: configFlags.Parse returned error: %v\n", err)
		log.Printf("[WARNING] Error parsing command-line flags: %v", err)
		return cfg, fmt.Errorf("error parsing flags: %w", err)
	}

	// Apply CLI overrides
	userSet := make(map[string]bool)
	configFlags.Visit(func(f *flag.Flag) {
		userSet[f.Name] = true
	})

	if userSet["topicIndexDir"] {
		cfg.TopicIndexDir = *cliTopicIndexDir
		log.Printf("[INFO] TopicIndexDir overridden by CLI flag: %s", cfg.TopicIndexDir)
	}
	if userSet["subForumListFile"] {
		cfg.SubForumListFile = *cliSubForumListFile
		log.Printf("[INFO] SubForumListFile overridden by CLI flag: %s", cfg.SubForumListFile)
	}
	if userSet["topicIndexFilePattern"] {
		cfg.TopicIndexFilePattern = *cliTopicIndexFilePattern
		log.Printf("[INFO] TopicIndexFilePattern overridden by CLI flag: %s", cfg.TopicIndexFilePattern)
	}
	if userSet["politenessDelay"] {
		parsedDuration, err := time.ParseDuration(*cliPolitenessDelay)
		if err != nil {
			log.Printf("[WARNING] Invalid politenessDelay format from CLI '%s': %v. Using previous value: %s", *cliPolitenessDelay, err, cfg.PolitenessDelay)
		} else {
			cfg.PolitenessDelay = parsedDuration
			log.Printf("[INFO] PolitenessDelay overridden by CLI flag: %s", cfg.PolitenessDelay)
		}
	}
	if userSet["userAgent"] {
		cfg.UserAgent = *cliUserAgent
		log.Printf("[INFO] UserAgent overridden by CLI flag: %s", cfg.UserAgent)
	}
	if userSet["archiveRootDir"] {
		cfg.ArchiveRootDir = *cliArchiveRootDir
		log.Printf("[INFO] ArchiveRootDir overridden by CLI flag: %s", cfg.ArchiveRootDir)
	}
	if userSet["stateFilePath"] {
		cfg.StateFilePath = *cliStateFilePath
		log.Printf("[INFO] StateFilePath overridden by CLI flag: %s", cfg.StateFilePath)
	}
	if userSet["saveStateInterval"] {
		parsedDuration, err := time.ParseDuration(*cliSaveStateInterval)
		if err != nil {
			log.Printf("[WARNING] Invalid saveStateInterval format from CLI '%s': %v. Using previous value: %s", *cliSaveStateInterval, err, cfg.SaveStateInterval)
		} else {
			cfg.SaveStateInterval = parsedDuration
			log.Printf("[INFO] SaveStateInterval overridden by CLI flag: %s", cfg.SaveStateInterval)
		}
	}
	if userSet["performanceLogPath"] {
		cfg.PerformanceLogPath = *cliPerformanceLogPath
		log.Printf("[INFO] PerformanceLogPath overridden by CLI flag: %s", cfg.PerformanceLogPath)
	}
	if userSet["logLevel"] {
		cfg.LogLevel = strings.ToUpper(*cliLogLevel) // Normalize to uppercase
		log.Printf("[INFO] LogLevel overridden by CLI flag: %s", cfg.LogLevel)
	}
	if userSet["logFilePath"] {
		cfg.LogFilePath = *cliLogFilePath
		log.Printf("[INFO] LogFilePath overridden by CLI flag: %s", cfg.LogFilePath)
	}
	if userSet["jitRefreshPages"] {
		cfg.JITRefreshPages = *cliJITRefreshPages
		log.Printf("[INFO] JITRefreshPages overridden by CLI flag: %d", cfg.JITRefreshPages)
	}
	if userSet["jitRefreshInterval"] {
		parsedDuration, err := time.ParseDuration(*cliJITRefreshInterval)
		if err != nil {
			log.Printf("[WARNING] Invalid jitRefreshInterval format from CLI '%s': %v. Using previous value: %s", *cliJITRefreshInterval, err, cfg.JITRefreshInterval)
		} else {
			cfg.JITRefreshInterval = parsedDuration
			log.Printf("[INFO] JITRefreshInterval overridden by CLI flag: %s", cfg.JITRefreshInterval)
		}
	}
	if userSet["forumBaseURL"] {
		cfg.ForumBaseURL = *cliForumBaseURL
		log.Printf("[INFO] ForumBaseURL overridden by CLI flag: %s", cfg.ForumBaseURL)
	}
	if userSet["archiveOutputRootDir"] {
		cfg.ArchiveOutputRootDir = *cliArchiveOutputRootDir
		log.Printf("[INFO] ArchiveOutputRootDir overridden by CLI flag: %s", cfg.ArchiveOutputRootDir)
	}

	// log.Printf("[DEBUG] config.LoadConfig: Skipping final CLI flag parsing. Current cfg.SubForumListFile: %s", cfg.SubForumListFile)

	return cfg, nil
}

// loadFromEnv loads configuration from environment variables.
// It modifies the passed-in cfg object directly.
// Environment variables should be prefixed (e.g., WAYPOINT_ARCHIVER_).
func loadFromEnv(cfg *Config) {
	// Helper to get env var only if it exists AND is non-empty
	loadStrEnv := func(envKey string, currentVal string) string {
		if value, exists := os.LookupEnv(envKey); exists && value != "" {
			log.Printf("[INFO] Loading '%s' from environment variable '%s'", envKey, value)
			return value
		}
		return currentVal // Return current (default/from file) if env var not set or is empty
	}

	loadIntEnv := func(envKey string, currentVal int) int {
		if valStr, exists := os.LookupEnv(envKey); exists && valStr != "" {
			valInt, err := strconv.Atoi(valStr)
			if err == nil {
				log.Printf("[INFO] Loading '%s' from environment variable '%s'", envKey, valStr)
				return valInt
			}
			log.Printf("[WARNING] Invalid integer format for env var %s='%s': %v. Using previous value: %d", envKey, valStr, err, currentVal)
		}
		return currentVal // Return current if env var not set, empty, or invalid format
	}

	loadDurationEnv := func(envKey string, currentVal time.Duration) time.Duration {
		if valStr, exists := os.LookupEnv(envKey); exists && valStr != "" {
			parsedDuration, err := time.ParseDuration(valStr)
			if err == nil {
				log.Printf("[INFO] Loading '%s' from environment variable '%s'", envKey, valStr)
				return parsedDuration
			}
			log.Printf("[WARNING] Invalid duration format for env var %s='%s': %v. Using previous value: %s", envKey, valStr, err, currentVal)
		}
		return currentVal // Return current if env var not set, empty, or invalid format
	}

	cfg.TopicIndexDir = loadStrEnv("WAYPOINT_TOPIC_INDEX_DIR", cfg.TopicIndexDir)
	cfg.SubForumListFile = loadStrEnv("WAYPOINT_SUBFORUM_LIST_FILE", cfg.SubForumListFile)
	cfg.TopicIndexFilePattern = loadStrEnv("WAYPOINT_TOPIC_INDEX_FILE_PATTERN", cfg.TopicIndexFilePattern)
	cfg.PolitenessDelay = loadDurationEnv("WAYPOINT_POLITENESS_DELAY", cfg.PolitenessDelay)
	cfg.UserAgent = loadStrEnv("WAYPOINT_USER_AGENT", cfg.UserAgent)
	cfg.ArchiveRootDir = loadStrEnv("WAYPOINT_ARCHIVE_ROOT_DIR", cfg.ArchiveRootDir)                    // Used by storer
	cfg.ArchiveOutputRootDir = loadStrEnv("WAYPOINT_ARCHIVE_OUTPUT_ROOT_DIR", cfg.ArchiveOutputRootDir) // Added for this task
	cfg.StateFilePath = loadStrEnv("WAYPOINT_STATE_FILE_PATH", cfg.StateFilePath)
	cfg.PerformanceLogPath = loadStrEnv("WAYPOINT_PERFORMANCE_LOG_PATH", cfg.PerformanceLogPath)
	cfg.JITRefreshPages = loadIntEnv("WAYPOINT_JIT_REFRESH_PAGES", cfg.JITRefreshPages)
	cfg.JITRefreshInterval = loadDurationEnv("WAYPOINT_JIT_REFRESH_INTERVAL", cfg.JITRefreshInterval)
	cfg.LogFilePath = loadStrEnv("WAYPOINT_LOG_FILE_PATH", cfg.LogFilePath) // Handles empty string correctly by design
	cfg.ForumBaseURL = loadStrEnv("WAYPOINT_FORUM_BASE_URL", cfg.ForumBaseURL)
	cfg.SaveStateInterval = loadDurationEnv("WAYPOINT_SAVE_STATE_INTERVAL", cfg.SaveStateInterval)

	// Handle LogLevel with validation
	if logLevelStr, exists := os.LookupEnv("WAYPOINT_LOG_LEVEL"); exists && logLevelStr != "" {
		normalizedLevel := strings.ToUpper(logLevelStr)
		switch normalizedLevel {
		case "DEBUG", "INFO", "WARN", "ERROR":
			log.Printf("[INFO] Loading 'WAYPOINT_LOG_LEVEL' from environment variable '%s'", logLevelStr)
			cfg.LogLevel = normalizedLevel
		default:
			log.Printf("[WARNING] Invalid LogLevel '%s' from env var WAYPOINT_LOG_LEVEL. Using previous value: %s", logLevelStr, cfg.LogLevel)
		}
	}

	// Load TestSubForumIDs from ENV (comma-separated string)
	if valStr, exists := os.LookupEnv("WAYPOINT_TEST_SUB_FORUM_IDS"); exists && valStr != "" {
		cfg.TestSubForumIDs = strings.Split(valStr, ",")
		log.Printf("[INFO] TestSubForumIDs loaded from environment variable WAYPOINT_TEST_SUB_FORUM_IDS: %v", cfg.TestSubForumIDs)
	}
}

// Helper to create a dummy config file for testing
func CreateDummyConfigFile(path string, content Config) error {
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Helper to remove a dummy config file
func RemoveDummyConfigFile(path string) error {
	err := os.Remove(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
