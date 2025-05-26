package config

import (
	"log"
	"os"

	//"path/filepath" // Not actually used, can remove if not needed later
	"flag" // No longer directly needed here for global manipulation
	"strings"
	"testing"
	"time"
)

// Remove setTestFlags as LoadConfig now takes args directly
// func setTestFlags(args []string) { ... }

func TestLoadConfig_Defaults(t *testing.T) {
	// Ensure no config.json exists for this test
	_ = RemoveDummyConfigFile(configFile) // best effort removal

	cfg, err := LoadConfig(nil) // Pass nil or empty slice for no CLI args
	if err != nil {
		t.Fatalf("LoadConfig(nil) error = %v, wantErr nil", err)
	}

	defaults := DefaultConfig()
	if cfg.TopicIndexDir != defaults.TopicIndexDir {
		t.Errorf("TopicIndexDir got = %s, want %s", cfg.TopicIndexDir, defaults.TopicIndexDir)
	}
	if cfg.SubForumListFile != defaults.SubForumListFile {
		t.Errorf("SubForumListFile got = %s, want %s", cfg.SubForumListFile, defaults.SubForumListFile)
	}
	if cfg.PolitenessDelay != defaults.PolitenessDelay {
		t.Errorf("PolitenessDelay got = %v, want %v", cfg.PolitenessDelay, defaults.PolitenessDelay)
	}
	if cfg.UserAgent != defaults.UserAgent {
		t.Errorf("UserAgent got = %s, want %s", cfg.UserAgent, defaults.UserAgent)
	}
	if cfg.ArchiveRootDir != defaults.ArchiveRootDir {
		t.Errorf("ArchiveRootDir got = %s, want %s", cfg.ArchiveRootDir, defaults.ArchiveRootDir)
	}
	if cfg.ArchiveOutputRootDir != defaults.ArchiveOutputRootDir {
		t.Errorf("ArchiveOutputRootDir got = %s, want %s", cfg.ArchiveOutputRootDir, defaults.ArchiveOutputRootDir)
	}
	if cfg.StateFilePath != defaults.StateFilePath {
		t.Errorf("StateFilePath got = %s, want %s", cfg.StateFilePath, defaults.StateFilePath)
	}
	if cfg.PerformanceLogPath != defaults.PerformanceLogPath {
		t.Errorf("PerformanceLogPath got = %s, want %s", cfg.PerformanceLogPath, defaults.PerformanceLogPath)
	}
	if cfg.LogLevel != defaults.LogLevel {
		t.Errorf("LogLevel got = %s, want %s", cfg.LogLevel, defaults.LogLevel)
	}
	if cfg.LogFilePath != defaults.LogFilePath {
		t.Errorf("LogFilePath got = %s, want %s", cfg.LogFilePath, defaults.LogFilePath)
	}
	if cfg.JITRefreshPages != defaults.JITRefreshPages {
		t.Errorf("JITRefreshPages got = %d, want %d", cfg.JITRefreshPages, defaults.JITRefreshPages)
	}
	if cfg.ConfigFilePath != defaults.ConfigFilePath { // Though not serialized, it's set by default
		t.Errorf("ConfigFilePath got = %s, want %s", cfg.ConfigFilePath, defaults.ConfigFilePath)
	}
	if cfg.ForumBaseURL != defaults.ForumBaseURL {
		t.Errorf("ForumBaseURL got = %s, want %s", cfg.ForumBaseURL, defaults.ForumBaseURL)
	}
}

func TestLoadConfig_ConfigFile(t *testing.T) {
	dummyContent := Config{
		TopicIndexDir:        "test_topics_from_config",
		SubForumListFile:     "test_subforums_from_config.csv",
		PolitenessDelay:      5 * time.Second,
		UserAgent:            "TestAgentFromConfig/1.0",
		ArchiveRootDir:       "test_archive_from_config",
		ArchiveOutputRootDir: "test_output_dir_from_config",
		StateFilePath:        "test_statefile_from_config.json",
		PerformanceLogPath:   "test_perflog_from_config.csv",
		LogLevel:             "DEBUG",
		LogFilePath:          "/logs/test_config.log",
		JITRefreshPages:      5,
		ForumBaseURL:         "http://config.example.com",
	}

	if err := CreateDummyConfigFile(configFile, dummyContent); err != nil {
		t.Fatalf("Failed to create dummy config file %s: %v", configFile, err)
	}
	defer RemoveDummyConfigFile(configFile)

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig(nil) with config file error = %v, wantErr nil", err)
	}

	if cfg.TopicIndexDir != dummyContent.TopicIndexDir {
		t.Errorf("TopicIndexDir got = %s, want %s", cfg.TopicIndexDir, dummyContent.TopicIndexDir)
	}
	if cfg.SubForumListFile != dummyContent.SubForumListFile {
		t.Errorf("SubForumListFile got = %s, want %s", cfg.SubForumListFile, dummyContent.SubForumListFile)
	}
	if cfg.PolitenessDelay != dummyContent.PolitenessDelay {
		t.Errorf("PolitenessDelay got = %v, want %v", cfg.PolitenessDelay, dummyContent.PolitenessDelay)
	}
	if cfg.UserAgent != dummyContent.UserAgent {
		t.Errorf("UserAgent got = %s, want %s", cfg.UserAgent, dummyContent.UserAgent)
	}
	if cfg.ArchiveRootDir != dummyContent.ArchiveRootDir {
		t.Errorf("ArchiveRootDir got = %s, want %s", cfg.ArchiveRootDir, dummyContent.ArchiveRootDir)
	}
	if cfg.ArchiveOutputRootDir != dummyContent.ArchiveOutputRootDir {
		t.Errorf("ArchiveOutputRootDir got = %s, want %s", cfg.ArchiveOutputRootDir, dummyContent.ArchiveOutputRootDir)
	}
	if cfg.StateFilePath != dummyContent.StateFilePath {
		t.Errorf("StateFilePath got = %s, want %s", cfg.StateFilePath, dummyContent.StateFilePath)
	}
	if cfg.PerformanceLogPath != dummyContent.PerformanceLogPath {
		t.Errorf("PerformanceLogPath got = %s, want %s", cfg.PerformanceLogPath, dummyContent.PerformanceLogPath)
	}
	if cfg.LogLevel != dummyContent.LogLevel {
		t.Errorf("LogLevel got = %s, want %s", cfg.LogLevel, dummyContent.LogLevel)
	}
	if cfg.LogFilePath != dummyContent.LogFilePath {
		t.Errorf("LogFilePath got = %s, want %s", cfg.LogFilePath, dummyContent.LogFilePath)
	}
	if cfg.JITRefreshPages != dummyContent.JITRefreshPages {
		t.Errorf("JITRefreshPages got = %d, want %d", cfg.JITRefreshPages, dummyContent.JITRefreshPages)
	}
	if cfg.ForumBaseURL != dummyContent.ForumBaseURL {
		t.Errorf("ForumBaseURL got = %s, want %s", cfg.ForumBaseURL, dummyContent.ForumBaseURL)
	}
	if cfg.ConfigFilePath != DefaultConfig().ConfigFilePath {
		t.Errorf("ConfigFilePath got = %s, want %s", cfg.ConfigFilePath, DefaultConfig().ConfigFilePath)
	}
}

func TestLoadConfig_CliOverrides(t *testing.T) {
	// Ensure no config.json for a clean CLI override test vs defaults
	_ = RemoveDummyConfigFile(configFile)
	defaults := DefaultConfig() // Get a fresh default config for comparisons

	tests := []struct {
		name                 string
		args                 []string
		wantTopicIndexDir    string
		wantSubForumListFile string
		wantDelay            time.Duration
		wantUA               string
		wantArchive          string
		wantArchiveOutputDir string
		wantStateFile        string
		wantPerfLogPath      string
		wantLogLevel         string
		wantLogFilePath      string
		wantJITPages         int
		wantForumBaseURL     string
		wantErr              bool   // For errors from LoadConfig itself (e.g. flag parsing error like -help)
		wantLog              string // Substring to expect in log for bad flag value parse by LoadConfig
	}{
		{
			name:                 "CLI PolitenessDelay override",
			args:                 []string{"-politenessDelay=10s"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            10 * time.Second,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
		{
			name:                 "CLI UserAgent override",
			args:                 []string{"-userAgent=TestAgentFromCLI/1.0"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               "TestAgentFromCLI/1.0",
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
		{
			name:                 "CLI ArchiveRootDir override",
			args:                 []string{"-archiveRootDir=cli_archive_dir"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               defaults.UserAgent,
			wantArchive:          "cli_archive_dir",
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
		{
			name:                 "CLI StateFilePath override",
			args:                 []string{"-stateFilePath=cli_state.json"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        "cli_state.json",
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
		{
			name:                 "CLI PerformanceLogPath override",
			args:                 []string{"-performanceLogPath=cli_perf.log"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      "cli_perf.log",
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
		{
			name:                 "CLI LogLevel override",
			args:                 []string{"-logLevel=DEBUG"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         "DEBUG",
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
		{
			name:                 "CLI LogFilePath override",
			args:                 []string{"-logFilePath=/tmp/cli.log"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      "/tmp/cli.log",
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
		{
			name:                 "CLI JITRefreshPages override",
			args:                 []string{"-jitRefreshPages=10"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         10,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
		{
			name:                 "CLI ArchiveOutputRootDir override",
			args:                 []string{"-archiveOutputRootDir=cli_output_dir"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: "cli_output_dir",
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
		{
			name: "All CLI overrides",
			args: []string{
				"-topicIndexDir=cli_topics",
				"-subForumListFile=cli_subforums.csv",
				"-politenessDelay=15s",
				"-userAgent=SuperTestAgentCLI/2.0",
				"-archiveRootDir=cli_main_archive",
				"-archiveOutputRootDir=cli_specific_output",
				"-stateFilePath=cli_archiver_state.json",
				"-performanceLogPath=cli_perf_log.csv",
				"-logLevel=warn",
				"-logFilePath=/tmp/archiver_cli.log",
				"-jitRefreshPages=7",
				"-forumBaseURL=http://cli.forum.com",
			},
			wantTopicIndexDir:    "cli_topics",
			wantSubForumListFile: "cli_subforums.csv",
			wantDelay:            15 * time.Second,
			wantUA:               "SuperTestAgentCLI/2.0",
			wantArchive:          "cli_main_archive",
			wantArchiveOutputDir: "cli_specific_output",
			wantStateFile:        "cli_archiver_state.json",
			wantPerfLogPath:      "cli_perf_log.csv",
			wantLogLevel:         "WARN",
			wantLogFilePath:      "/tmp/archiver_cli.log",
			wantJITPages:         7,
			wantForumBaseURL:     "http://cli.forum.com",
		},
		{
			name:                 "CLI invalid politenessDelay format",
			args:                 []string{"-politenessDelay=invalid"},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay, // Should remain default
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
			wantErr:              false, // The LoadConfig itself doesn't error, just logs warning and uses previous value
			wantLog:              "Invalid politenessDelay format from CLI",
		},
		{
			name:    "CLI help flag",
			args:    []string{"-help"},
			wantErr: true, // LoadConfig should return flag.ErrHelp
			// Set other wants to defaults, though they won't be checked if wantErr is true
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf strings.Builder
			log.SetOutput(&logBuf)
			defer log.SetOutput(os.Stderr)

			cfg, err := LoadConfig(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadConfig(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			// If we expect an error (like -help), don't proceed to check values
			if tt.wantErr {
				if err != flag.ErrHelp { // Be more specific if expecting help error
					// t.Logf("Got error: %v", err) // Potentially log for debugging non-help errors
				}
				return
			}

			if cfg.TopicIndexDir != tt.wantTopicIndexDir {
				t.Errorf("TopicIndexDir got = %s, want %s", cfg.TopicIndexDir, tt.wantTopicIndexDir)
			}
			if cfg.SubForumListFile != tt.wantSubForumListFile {
				t.Errorf("SubForumListFile got = %s, want %s", cfg.SubForumListFile, tt.wantSubForumListFile)
			}
			if cfg.PolitenessDelay != tt.wantDelay {
				t.Errorf("PolitenessDelay got = %v, want %v", cfg.PolitenessDelay, tt.wantDelay)
			}
			if cfg.UserAgent != tt.wantUA {
				t.Errorf("UserAgent got = %s, want %s", cfg.UserAgent, tt.wantUA)
			}
			if cfg.ArchiveRootDir != tt.wantArchive {
				t.Errorf("ArchiveRootDir got = %s, want %s", cfg.ArchiveRootDir, tt.wantArchive)
			}
			if cfg.ArchiveOutputRootDir != tt.wantArchiveOutputDir {
				t.Errorf("ArchiveOutputRootDir got = %s, want %s", cfg.ArchiveOutputRootDir, tt.wantArchiveOutputDir)
			}
			if cfg.StateFilePath != tt.wantStateFile {
				t.Errorf("StateFilePath got = %s, want %s", cfg.StateFilePath, tt.wantStateFile)
			}
			if cfg.PerformanceLogPath != tt.wantPerfLogPath {
				t.Errorf("PerformanceLogPath got = %s, want %s", cfg.PerformanceLogPath, tt.wantPerfLogPath)
			}
			if cfg.LogLevel != tt.wantLogLevel {
				t.Errorf("LogLevel got = %s, want %s", cfg.LogLevel, tt.wantLogLevel)
			}
			if cfg.LogFilePath != tt.wantLogFilePath {
				t.Errorf("LogFilePath got = %s, want %s", cfg.LogFilePath, tt.wantLogFilePath)
			}
			if cfg.JITRefreshPages != tt.wantJITPages {
				t.Errorf("JITRefreshPages got = %d, want %d", cfg.JITRefreshPages, tt.wantJITPages)
			}
			if cfg.ForumBaseURL != tt.wantForumBaseURL {
				t.Errorf("ForumBaseURL got = %s, want %s", cfg.ForumBaseURL, tt.wantForumBaseURL)
			}

			if tt.wantLog != "" && !strings.Contains(logBuf.String(), tt.wantLog) {
				t.Errorf("Expected log to contain '%s', got: %s", tt.wantLog, logBuf.String())
			}
		})
	}
}

func TestLoadConfig_ConfigFileAndCliOverrides(t *testing.T) {
	defaults := DefaultConfig()
	configFileContent := Config{
		TopicIndexDir:        "config_topics",
		SubForumListFile:     "config_subforums.csv",
		PolitenessDelay:      5 * time.Second,
		UserAgent:            "ConfigAgent/1.0",
		ArchiveRootDir:       "config_archive",
		ArchiveOutputRootDir: "config_archive_output",
		StateFilePath:        "config_state.json",
		PerformanceLogPath:   "config_perf.log",
		LogLevel:             "INFO",
		LogFilePath:          "config_app.log",
		JITRefreshPages:      2,
		ForumBaseURL:         "http://file.forum.com",
	}

	if err := CreateDummyConfigFile(configFile, configFileContent); err != nil {
		t.Fatalf("Failed to create dummy config file: %v", err)
	}
	defer RemoveDummyConfigFile(configFile)

	cliArgs := []string{
		"-politenessDelay=15s",
		"-userAgent=OverrideAgentFromCLI/2.0",
		"-stateFilePath=override_cli_state.json",
		"-logLevel=ERROR",
		"-jitRefreshPages=22",
		"-archiveOutputRootDir=cli_out_final",
	}

	// Define and manage environment variables for this test
	envVarsToSet := map[string]string{
		"WAYPOINT_LOG_LEVEL":               "ENV_DEBUG",
		"WAYPOINT_POLITENESS_DELAY":        "100s",
		"WAYPOINT_LOG_FILE_PATH":           "/env/override.log",
		"WAYPOINT_ARCHIVE_ROOT_DIR":        "/env_only_archive_root/",
		"WAYPOINT_ARCHIVE_OUTPUT_ROOT_DIR": "env_output_dir_should_be_overridden_by_config_then_cli",
	}
	originalEnvValues := make(map[string]string)
	for k, v := range envVarsToSet {
		if oldVal, exists := os.LookupEnv(k); exists {
			originalEnvValues[k] = oldVal
		}
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVarsToSet {
			if oldVal, exists := originalEnvValues[k]; exists {
				os.Setenv(k, oldVal)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	cfg, err := LoadConfig(cliArgs)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, wantErr nil", err)
	}

	// Assertions: Check precedence
	// CLI > Config File > Environment Variables > Defaults

	// CLI should override config file and env vars for specified flags
	if cfg.PolitenessDelay != 15*time.Second {
		t.Errorf("PolitenessDelay got = %v, want %v (CLI override)", cfg.PolitenessDelay, 15*time.Second)
	}
	if cfg.UserAgent != "OverrideAgentFromCLI/2.0" {
		t.Errorf("UserAgent got = %s, want %s (CLI override)", cfg.UserAgent, "OverrideAgentFromCLI/2.0")
	}
	if cfg.StateFilePath != "override_cli_state.json" {
		t.Errorf("StateFilePath got = %s, want %s (CLI override)", cfg.StateFilePath, "override_cli_state.json")
	}
	if cfg.LogLevel != "ERROR" {
		t.Errorf("LogLevel got = %s, want ERROR (CLI override)", cfg.LogLevel)
	}
	if cfg.JITRefreshPages != 22 {
		t.Errorf("JITRefreshPages got = %d, want 22 (CLI override)", cfg.JITRefreshPages)
	}
	if cfg.ArchiveOutputRootDir != "cli_out_final" {
		t.Errorf("ArchiveOutputRootDir got = %s, want cli_out_final (from CLI)", cfg.ArchiveOutputRootDir)
	}

	// Values from config file (should override initial defaults and initial ENV, but be overridden by CLI if specified)
	if cfg.TopicIndexDir != configFileContent.TopicIndexDir {
		t.Errorf("TopicIndexDir got = %s, want %s (from config file)", cfg.TopicIndexDir, configFileContent.TopicIndexDir)
	}
	if cfg.SubForumListFile != configFileContent.SubForumListFile {
		t.Errorf("SubForumListFile got = %s, want %s (from config file)", cfg.SubForumListFile, configFileContent.SubForumListFile)
	}
	if cfg.PerformanceLogPath != configFileContent.PerformanceLogPath {
		t.Errorf("PerformanceLogPath got = %s, want %s (from config file)", cfg.PerformanceLogPath, configFileContent.PerformanceLogPath)
	}

	// Value from ENV (WAYPOINT_ARCHIVE_ROOT_DIR) because not in CLI, but present in ENV (this ENV value should override the one in config file due to second loadFromEnv call)
	if cfg.ArchiveRootDir != "/env_only_archive_root/" {
		t.Errorf("ArchiveRootDir got = %s, want %s (from ENV override of config)", cfg.ArchiveRootDir, "/env_only_archive_root/")
	}

	// Value for LogFilePath: Config has "config_app.log", ENV has "/env/override.log". ENV (from second loadFromEnv) should win.
	if cfg.LogFilePath != "/env/override.log" {
		t.Errorf("LogFilePath got = %s, want %s (from ENV override of config)", cfg.LogFilePath, "/env/override.log")
	}

	// ConfigFilePath should be the default as CLI did not specify a different one.
	if cfg.ConfigFilePath != defaults.ConfigFilePath {
		t.Errorf("ConfigFilePath got = %s, want %s", cfg.ConfigFilePath, defaults.ConfigFilePath)
	}

	// ForumBaseURL should come from config file, as it's not set by CLI or ENV in this specific sub-test setup
	if cfg.ForumBaseURL != configFileContent.ForumBaseURL {
		t.Errorf("ForumBaseURL got = %s, want %s (from config)", cfg.ForumBaseURL, configFileContent.ForumBaseURL)
	}
}

func TestLoadConfig_MalformedConfigFile(t *testing.T) {
	malformedJSON := "{\"politenessDelay\": \"5s\", \"userAgent\": \"MalformedAgent/1.0\" // missing closing brace"
	if err := os.WriteFile(configFile, []byte(malformedJSON), 0644); err != nil {
		t.Fatalf("Failed to write malformed config file: %v", err)
	}
	defer RemoveDummyConfigFile(configFile)

	var logBuf strings.Builder
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	cfg, err := LoadConfig(nil)
	if err != nil {
		// This specific error might occur if unmarshalling completely fails and returns a default cfg + error
		// For now, the behavior is to log and proceed with defaults. So LoadConfig error should be nil.
		t.Fatalf("LoadConfig(nil) with malformed config error = %v, wantErr nil or specific parse error", err)
	}

	// Should log a warning and use defaults
	if !strings.Contains(logBuf.String(), "Error unmarshalling config file") {
		t.Errorf("Expected log to contain unmarshalling error, got: %s", logBuf.String())
	}

	defaults := DefaultConfig()
	if cfg.PolitenessDelay != defaults.PolitenessDelay {
		t.Errorf("PolitenessDelay got = %v, want default %v after malformed config", cfg.PolitenessDelay, defaults.PolitenessDelay)
	}
}

// TestLoadConfig_EnvVarsOnly tests loading configuration solely from environment variables.
func TestLoadConfig_EnvVarsOnly(t *testing.T) {
	_ = RemoveDummyConfigFile(configFile) // Ensure no config file interference
	defaults := DefaultConfig()

	tests := []struct {
		name                 string
		envVars              map[string]string
		wantTopicIndexDir    string
		wantSubForumListFile string
		wantDelay            time.Duration
		wantUA               string
		wantArchive          string
		wantArchiveOutputDir string
		wantStateFile        string
		wantPerfLogPath      string
		wantLogLevel         string
		wantLogFilePath      string
		wantJITPages         int
		wantForumBaseURL     string
		wantLogContains      []string
	}{
		{
			name: "Specific Env Vars",
			envVars: map[string]string{
				"WAYPOINT_LOG_LEVEL":         "DEBUG",
				"WAYPOINT_JIT_REFRESH_PAGES": "7",
				"WAYPOINT_POLITENESS_DELAY":  "750ms",
			},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            750 * time.Millisecond,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         "DEBUG",
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         7,
			wantForumBaseURL:     defaults.ForumBaseURL,
			wantLogContains: []string{
				"Loading 'WAYPOINT_LOG_LEVEL' from environment variable",
				"Loading 'WAYPOINT_JIT_REFRESH_PAGES' from environment variable",
				"Loading 'WAYPOINT_POLITENESS_DELAY' from environment variable",
			},
		},
		{
			name: "All Env Vars",
			envVars: map[string]string{
				"WAYPOINT_TOPIC_INDEX_DIR":         "/env/topics_all",
				"WAYPOINT_SUBFORUM_LIST_FILE":      "/env/subforums_all.csv",
				"WAYPOINT_POLITENESS_DELAY":        "1234ms",
				"WAYPOINT_USER_AGENT":              "EnvAgent_All/0.1",
				"WAYPOINT_ARCHIVE_ROOT_DIR":        "/env/archive_out_all",
				"WAYPOINT_ARCHIVE_OUTPUT_ROOT_DIR": "/env/output_dir_all",
				"WAYPOINT_STATE_FILE_PATH":         "/env/state_all.json",
				"WAYPOINT_PERFORMANCE_LOG_PATH":    "/env/perf_all.csv",
				"WAYPOINT_LOG_LEVEL":               "warn",
				"WAYPOINT_LOG_FILE_PATH":           "/env/app_all.log",
				"WAYPOINT_JIT_REFRESH_PAGES":       "12",
				"WAYPOINT_FORUM_BASE_URL":          "http://env-all.forum.com",
			},
			wantTopicIndexDir:    "/env/topics_all",
			wantSubForumListFile: "/env/subforums_all.csv",
			wantDelay:            1234 * time.Millisecond,
			wantUA:               "EnvAgent_All/0.1",
			wantArchive:          "/env/archive_out_all",
			wantArchiveOutputDir: "/env/output_dir_all",
			wantStateFile:        "/env/state_all.json",
			wantPerfLogPath:      "/env/perf_all.csv",
			wantLogLevel:         "WARN",
			wantLogFilePath:      "/env/app_all.log",
			wantJITPages:         12,
			wantForumBaseURL:     "http://env-all.forum.com",
			wantLogContains: []string{
				"Loading 'WAYPOINT_TOPIC_INDEX_DIR'",
				"Loading 'WAYPOINT_ARCHIVE_OUTPUT_ROOT_DIR'",
			},
		},
		{
			name: "Invalid Env Var formats",
			envVars: map[string]string{
				"WAYPOINT_POLITENESS_DELAY":  "invalid-duration",
				"WAYPOINT_JIT_REFRESH_PAGES": "not-an-int",
			},
			wantTopicIndexDir:    defaults.TopicIndexDir,
			wantSubForumListFile: defaults.SubForumListFile,
			wantDelay:            defaults.PolitenessDelay,
			wantUA:               defaults.UserAgent,
			wantArchive:          defaults.ArchiveRootDir,
			wantArchiveOutputDir: defaults.ArchiveOutputRootDir,
			wantStateFile:        defaults.StateFilePath,
			wantPerfLogPath:      defaults.PerformanceLogPath,
			wantLogLevel:         defaults.LogLevel,
			wantLogFilePath:      defaults.LogFilePath,
			wantJITPages:         defaults.JITRefreshPages,
			wantForumBaseURL:     defaults.ForumBaseURL,
			wantLogContains: []string{
				"Invalid duration format for env var WAYPOINT_POLITENESS_DELAY",
				"Invalid integer format for env var WAYPOINT_JIT_REFRESH_PAGES",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalEnvVars := make(map[string]string)
			for k, v := range tt.envVars {
				if oldVal, exists := os.LookupEnv(k); exists {
					originalEnvVars[k] = oldVal
				}
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					if oldVal, exists := originalEnvVars[k]; exists {
						os.Setenv(k, oldVal)
					} else {
						os.Unsetenv(k)
					}
				}
			}()

			var logBuf strings.Builder
			log.SetOutput(&logBuf)
			defer log.SetOutput(os.Stderr)

			cfg, err := LoadConfig(nil)
			if err != nil {
				t.Fatalf("LoadConfig(nil) with env vars error = %v, wantErr nil", err)
			}

			if cfg.TopicIndexDir != tt.wantTopicIndexDir {
				t.Errorf("TopicIndexDir got %s, want %s", cfg.TopicIndexDir, tt.wantTopicIndexDir)
			}
			if cfg.SubForumListFile != tt.wantSubForumListFile {
				t.Errorf("SubForumListFile got %s, want %s", cfg.SubForumListFile, tt.wantSubForumListFile)
			}
			if cfg.PolitenessDelay != tt.wantDelay {
				t.Errorf("PolitenessDelay got %v, want %v", cfg.PolitenessDelay, tt.wantDelay)
			}
			if cfg.UserAgent != tt.wantUA {
				t.Errorf("UserAgent got %s, want %s", cfg.UserAgent, tt.wantUA)
			}
			if cfg.ArchiveRootDir != tt.wantArchive {
				t.Errorf("ArchiveRootDir got %s, want %s", cfg.ArchiveRootDir, tt.wantArchive)
			}
			if cfg.ArchiveOutputRootDir != tt.wantArchiveOutputDir {
				t.Errorf("ArchiveOutputRootDir got %s, want %s", cfg.ArchiveOutputRootDir, tt.wantArchiveOutputDir)
			}
			if cfg.StateFilePath != tt.wantStateFile {
				t.Errorf("StateFilePath got %s, want %s", cfg.StateFilePath, tt.wantStateFile)
			}
			if cfg.PerformanceLogPath != tt.wantPerfLogPath {
				t.Errorf("PerformanceLogPath got %s, want %s", cfg.PerformanceLogPath, tt.wantPerfLogPath)
			}
			if cfg.LogLevel != tt.wantLogLevel {
				t.Errorf("LogLevel got %s, want %s", cfg.LogLevel, tt.wantLogLevel)
			}
			if cfg.LogFilePath != tt.wantLogFilePath {
				t.Errorf("LogFilePath got %s, want %s", cfg.LogFilePath, tt.wantLogFilePath)
			}
			if cfg.JITRefreshPages != tt.wantJITPages {
				t.Errorf("JITRefreshPages got %d, want %d", cfg.JITRefreshPages, tt.wantJITPages)
			}
			if cfg.ForumBaseURL != tt.wantForumBaseURL {
				t.Errorf("ForumBaseURL got %s, want %s", cfg.ForumBaseURL, tt.wantForumBaseURL)
			}

			actualLog := logBuf.String()
			for _, expectedLogFragment := range tt.wantLogContains {
				if !strings.Contains(actualLog, expectedLogFragment) {
					t.Errorf("Expected log to contain \"%s\", but it didn't. Log:\n%s", expectedLogFragment, actualLog)
				}
			}
		})
	}
}

func TestMain(m *testing.M) {
	// Clean up any dummy config file before and after tests
	_ = RemoveDummyConfigFile(configFile)
	code := m.Run()
	_ = RemoveDummyConfigFile(configFile)
	os.Exit(code)
}
