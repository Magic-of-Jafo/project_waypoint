package config

import "time"

// Config holds the application configuration.
// For now, it's basic. It can be expanded to load from a file or env vars.
type Config struct {
	TopicIndexDir    string        // Directory containing topic_index_*.csv files
	SubForumListFile string        // Path to the subforum_list.csv file
	PolitenessDelay  time.Duration // Delay between HTTP requests for politeness
}

// DefaultConfig returns a new Config with default values.
// These would typically be paths within a data directory.
func DefaultConfig() *Config {
	return &Config{
		TopicIndexDir:    "data/topic_indices",
		SubForumListFile: "data/subforum_list.csv",
		PolitenessDelay:  3 * time.Second, // Default politeness delay
	}
}

// LoadConfig loads the application configuration.
// Currently, it returns a default hardcoded configuration.
// This should be extended to load from a file (e.g., JSON, YAML) or environment variables.
func LoadConfig() (*Config, error) {
	// Default configuration for now
	// These paths will likely need to be relative to the project root or an absolute base path.
	// For Story 2.1, we're concerned with reading these. The actual location will be determined
	// by the user/environment setup.
	cfg := &Config{
		TopicIndexDir:    "./data/topic_indices",     // Example path
		SubForumListFile: "./data/subforum_list.csv", // Example path
	}
	return cfg, nil
}
