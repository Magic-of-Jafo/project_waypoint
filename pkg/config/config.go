package config

// Config holds the application configuration.
// For now, it primarily stores the path to the topic index data.
// This will be expanded for other configurable parameters.
type Config struct {
	TopicIndexDir    string // Directory containing topic_index_{subforum_id}.csv files
	SubForumListFile string // Path to data/subforum_list.csv
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
