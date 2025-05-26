package config

import (
	"log"
	"os"

	//"path/filepath" // Not actually used, can remove if not needed later
	"strings"
	"testing"
	"time"
	// "flag" // No longer directly needed here for global manipulation
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
	if cfg.PolitenessDelay != defaults.PolitenessDelay {
		t.Errorf("PolitenessDelay got = %v, want %v", cfg.PolitenessDelay, defaults.PolitenessDelay)
	}
	if cfg.UserAgent != defaults.UserAgent {
		t.Errorf("UserAgent got = %s, want %s", cfg.UserAgent, defaults.UserAgent)
	}
	if cfg.ArchiveRootDir != defaults.ArchiveRootDir {
		t.Errorf("ArchiveRootDir got = %s, want %s", cfg.ArchiveRootDir, defaults.ArchiveRootDir)
	}
	if cfg.StateFilePath != defaults.StateFilePath {
		t.Errorf("StateFilePath got = %s, want %s", cfg.StateFilePath, defaults.StateFilePath)
	}
}

func TestLoadConfig_ConfigFile(t *testing.T) {
	dummyContent := Config{
		PolitenessDelay:  5 * time.Second,
		UserAgent:        "TestAgentFromConfig/1.0",
		ArchiveRootDir:   "test_archive_from_config",
		TopicIndexDir:    "test_topics_from_config",
		SubForumListFile: "test_subforums_from_config.csv",
		StateFilePath:    "test_statefile_from_config.json",
	}

	if err := CreateDummyConfigFile(configFile, dummyContent); err != nil {
		t.Fatalf("Failed to create dummy config file %s: %v", configFile, err)
	}
	defer RemoveDummyConfigFile(configFile)

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig(nil) with config file error = %v, wantErr nil", err)
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
	if cfg.StateFilePath != dummyContent.StateFilePath {
		t.Errorf("StateFilePath got = %s, want %s", cfg.StateFilePath, dummyContent.StateFilePath)
	}
}

func TestLoadConfig_CliOverrides(t *testing.T) {
	// Ensure no config.json for a clean CLI override test vs defaults
	_ = RemoveDummyConfigFile(configFile)

	tests := []struct {
		name          string
		args          []string
		wantDelay     time.Duration
		wantUA        string
		wantArchive   string
		wantStateFile string
		wantErr       bool   // For errors from LoadConfig itself (e.g. flag parsing error like -help)
		wantLog       string // Substring to expect in log for bad flag value parse by LoadConfig
	}{
		{
			name:          "CLI PolitenessDelay override",
			args:          []string{"-politenessDelay=10s"},
			wantDelay:     10 * time.Second,
			wantUA:        DefaultConfig().UserAgent,
			wantArchive:   DefaultConfig().ArchiveRootDir,
			wantStateFile: DefaultConfig().StateFilePath,
		},
		{
			name:          "CLI UserAgent override",
			args:          []string{"-userAgent=TestAgentFromCLI/1.0"},
			wantDelay:     DefaultConfig().PolitenessDelay,
			wantUA:        "TestAgentFromCLI/1.0",
			wantArchive:   DefaultConfig().ArchiveRootDir,
			wantStateFile: DefaultConfig().StateFilePath,
		},
		{
			name:          "CLI ArchiveRootDir override",
			args:          []string{"-archiveRootDir=cli_archive_dir"},
			wantDelay:     DefaultConfig().PolitenessDelay,
			wantUA:        DefaultConfig().UserAgent,
			wantArchive:   "cli_archive_dir",
			wantStateFile: DefaultConfig().StateFilePath,
		},
		{
			name:          "CLI StateFilePath override",
			args:          []string{"-stateFilePath=cli_state.json"},
			wantDelay:     DefaultConfig().PolitenessDelay,
			wantUA:        DefaultConfig().UserAgent,
			wantArchive:   DefaultConfig().ArchiveRootDir,
			wantStateFile: "cli_state.json",
		},
		{
			name:          "CLI all overrides",
			args:          []string{"-politenessDelay=1s", "-userAgent=FullCLI/1.0", "-archiveRootDir=all_cli", "-stateFilePath=all_cli.json"},
			wantDelay:     1 * time.Second,
			wantUA:        "FullCLI/1.0",
			wantArchive:   "all_cli",
			wantStateFile: "all_cli.json",
		},
		{
			name:          "CLI invalid politenessDelay format",
			args:          []string{"-politenessDelay=invalid"},
			wantDelay:     DefaultConfig().PolitenessDelay,
			wantUA:        DefaultConfig().UserAgent,
			wantArchive:   DefaultConfig().ArchiveRootDir,
			wantStateFile: DefaultConfig().StateFilePath,
			wantErr:       false,
			wantLog:       "Invalid politenessDelay format from CLI",
		},
		{
			name:    "CLI help flag",
			args:    []string{"-help"},
			wantErr: true,
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
			if !tt.wantErr {
				if cfg.PolitenessDelay != tt.wantDelay {
					t.Errorf("PolitenessDelay got = %v, want %v", cfg.PolitenessDelay, tt.wantDelay)
				}
				if cfg.UserAgent != tt.wantUA {
					t.Errorf("UserAgent got = %s, want %s", cfg.UserAgent, tt.wantUA)
				}
				if cfg.ArchiveRootDir != tt.wantArchive {
					t.Errorf("ArchiveRootDir got = %s, want %s", cfg.ArchiveRootDir, tt.wantArchive)
				}
				if cfg.StateFilePath != tt.wantStateFile {
					t.Errorf("StateFilePath got = %s, want %s", cfg.StateFilePath, tt.wantStateFile)
				}
			}
			if tt.wantLog != "" && !strings.Contains(logBuf.String(), tt.wantLog) {
				t.Errorf("Expected log to contain '%s', got: %s", tt.wantLog, logBuf.String())
			}
		})
	}
}

func TestLoadConfig_ConfigFileAndCliOverrides(t *testing.T) {
	configFileContent := Config{
		PolitenessDelay:  5 * time.Second,
		UserAgent:        "ConfigAgent/1.0",
		ArchiveRootDir:   "config_archive",
		TopicIndexDir:    "config_topics",
		SubForumListFile: "config_subforums.csv",
		StateFilePath:    "config_state.json",
	}

	if err := CreateDummyConfigFile(configFile, configFileContent); err != nil {
		t.Fatalf("Failed to create dummy config file: %v", err)
	}
	defer RemoveDummyConfigFile(configFile)

	cliArgs := []string{
		"-politenessDelay=15s",
		"-userAgent=OverrideAgentFromCLI/2.0",
		"-stateFilePath=override_cli_state.json",
	}

	cfg, err := LoadConfig(cliArgs)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, wantErr nil", err)
	}

	wantDelay := 15 * time.Second
	wantUA := "OverrideAgentFromCLI/2.0"
	wantArchive := configFileContent.ArchiveRootDir
	wantStateFile := "override_cli_state.json"

	if cfg.PolitenessDelay != wantDelay {
		t.Errorf("PolitenessDelay got = %v, want %v", cfg.PolitenessDelay, wantDelay)
	}
	if cfg.UserAgent != wantUA {
		t.Errorf("UserAgent got = %s, want %s", cfg.UserAgent, wantUA)
	}
	if cfg.ArchiveRootDir != wantArchive {
		t.Errorf("ArchiveRootDir got = %s, want %s", cfg.ArchiveRootDir, wantArchive)
	}
	if cfg.StateFilePath != wantStateFile {
		t.Errorf("StateFilePath got = %s, want %s", cfg.StateFilePath, wantStateFile)
	}
	if cfg.TopicIndexDir != configFileContent.TopicIndexDir {
		t.Errorf("TopicIndexDir got = %s, want %s", cfg.TopicIndexDir, configFileContent.TopicIndexDir)
	}
}

func TestLoadConfig_MalformedConfigFile(t *testing.T) {
	malformedFilePath := configFile
	malformedContent := []byte("{\"politenessDelay\": \"5s\", userAgent: \"UnquotedKey\"}")

	if err := os.WriteFile(malformedFilePath, malformedContent, 0644); err != nil {
		t.Fatalf("Failed to create malformed config file: %v", err)
	}
	defer os.Remove(malformedFilePath)

	var logBuf strings.Builder
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() with malformed json error = %v, wantErr nil (should log warning)", err)
	}

	defaults := DefaultConfig()
	if cfg.PolitenessDelay != defaults.PolitenessDelay {
		t.Errorf("PolitenessDelay got = %v, want default %v after malformed config", cfg.PolitenessDelay, defaults.PolitenessDelay)
	}
	if !strings.Contains(logBuf.String(), "Error unmarshalling config file") {
		t.Errorf("Expected log warning about unmarshalling, got: %s", logBuf.String())
	}
}

func TestMain(m *testing.M) {
	defer RemoveDummyConfigFile(configFile)
	RemoveDummyConfigFile(configFile)

	code := m.Run()
	os.Exit(code)
}
