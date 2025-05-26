package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"time"
)

// Config holds the application configuration.
// For now, it's basic. It can be expanded to load from a file or env vars.
type Config struct {
	TopicIndexDir    string        `json:"topicIndexDir"`
	SubForumListFile string        `json:"subForumListFile"`
	PolitenessDelay  time.Duration `json:"politenessDelay"`
	UserAgent        string        `json:"userAgent"`
	ArchiveRootDir   string        `json:"archiveRootDir"` // Added for archiver, used by storer (Story 2.4)
}

// DefaultConfig returns a new Config with default values.
// These would typically be paths within a data directory.
func DefaultConfig() *Config {
	return &Config{
		TopicIndexDir:    "data/topic_indices",
		SubForumListFile: "data/subforum_list.csv",
		PolitenessDelay:  3 * time.Second,            // Default politeness delay
		UserAgent:        "WaypointArchiveAgent/1.0", // Default User-Agent
		ArchiveRootDir:   "archive_output",           // Default archive root
	}
}

const configFile = "config.json"

// LoadConfig loads the application configuration.
// It accepts command-line arguments (typically os.Args[1:]) for parsing.
// Order of precedence (lowest to highest):
// 1. Hardcoded default values.
// 2. Values from config.json file (if it exists).
// 3. Values from command-line flags (if provided and parsed from 'arguments').
func LoadConfig(arguments []string) (*Config, error) {
	// Start with hardcoded defaults
	cfg := DefaultConfig()

	// Attempt to load from config.json
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Printf("[WARNING] Error reading config file %s: %v. Using defaults/CLI flags.", configFile, err)
		} else {
			err = json.Unmarshal(data, cfg)
			if err != nil {
				log.Printf("[WARNING] Error unmarshalling config file %s: %v. Using defaults/CLI flags.", configFile, err)
			} else {
				log.Printf("[INFO] Loaded configuration from %s", configFile)
			}
		}
	} else if !os.IsNotExist(err) {
		log.Printf("[WARNING] Error checking for config file %s: %v. Using defaults/CLI flags.", configFile, err)
	}

	// Define and parse command-line flags using a local FlagSet
	configFlags := flag.NewFlagSet("config", flag.ContinueOnError) // ContinueOnError allows us to handle parsing errors

	cliPolitenessDelay := configFlags.String("politenessDelay", "", "Politeness delay (e.g., '3s', '500ms')")
	cliUserAgent := configFlags.String("userAgent", "", "Custom User-Agent string")
	cliArchiveRootDir := configFlags.String("archiveRootDir", "", "Root directory for storing archived files")
	cliTopicIndexDir := configFlags.String("topicIndexDir", "", "Directory for topic index CSVs")
	cliSubForumListFile := configFlags.String("subForumListFile", "", "Path to subforum list CSV")

	err := configFlags.Parse(arguments)
	if err != nil {
		// Log the error but don't necessarily fail catastrophically unless it's flag.ErrHelp
		// flag.ErrHelp is handled by the FlagSet's ErrorHandling policy (e.g., ExitOnError will exit)
		// If ContinueOnError, err will be returned and main can decide to print usage and exit.
		log.Printf("[WARNING] Error parsing command-line flags: %v", err)
		// We might still proceed with defaults/config file if parsing error is not fatal (e.g. ContinueOnError)
		// For now, let's return the error to let the caller decide.
		return cfg, fmt.Errorf("error parsing flags: %w", err) // Return error to caller
	}

	// Apply CLI overrides if flags were set
	userSet := make(map[string]bool)
	configFlags.Visit(func(f *flag.Flag) { // Visit only flags set by the user on this specific FlagSet
		userSet[f.Name] = true
	})

	if userSet["politenessDelay"] && *cliPolitenessDelay != "" {
		parsedDuration, err := time.ParseDuration(*cliPolitenessDelay)
		if err != nil {
			log.Printf("[WARNING] Invalid politenessDelay format from CLI '%s': %v. Using previous value: %s", *cliPolitenessDelay, err, cfg.PolitenessDelay)
		} else {
			cfg.PolitenessDelay = parsedDuration
			log.Printf("[INFO] PolitenessDelay overridden by CLI flag: %s", cfg.PolitenessDelay)
		}
	}

	if userSet["userAgent"] && *cliUserAgent != "" {
		cfg.UserAgent = *cliUserAgent
		log.Printf("[INFO] UserAgent overridden by CLI flag: %s", cfg.UserAgent)
	}

	if userSet["archiveRootDir"] && *cliArchiveRootDir != "" {
		cfg.ArchiveRootDir = *cliArchiveRootDir
		log.Printf("[INFO] ArchiveRootDir overridden by CLI flag: %s", cfg.ArchiveRootDir)
	}

	if userSet["topicIndexDir"] && *cliTopicIndexDir != "" {
		cfg.TopicIndexDir = *cliTopicIndexDir
		log.Printf("[INFO] TopicIndexDir overridden by CLI flag: %s", cfg.TopicIndexDir)
	}

	if userSet["subForumListFile"] && *cliSubForumListFile != "" {
		cfg.SubForumListFile = *cliSubForumListFile
		log.Printf("[INFO] SubForumListFile overridden by CLI flag: %s", cfg.SubForumListFile)
	}

	return cfg, nil
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
