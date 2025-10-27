package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds configuration for the password manager
type Config struct {
	StoreDir     string
	Editor       string
	GPGKeyID     string
	CacheTimeout int    // Cache timeout in minutes
	GitURL       string // Git repository URL for sync
	GitAutoSync  bool   // Automatically sync changes to Git
}

// Load loads configuration from environment variables and defaults
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Default configuration
	cfg := &Config{
		StoreDir:     filepath.Join(homeDir, ".password-store"),
		Editor:       getEnvDefault("EDITOR", "vim"),
		CacheTimeout: 5,    // Default 5 minutes
		GitAutoSync:  true, // Auto-sync enabled by default
	}

	// Override with environment variables if set
	if storeDir := os.Getenv("PASSWORD_STORE_DIR"); storeDir != "" {
		cfg.StoreDir = storeDir
	}

	if gpgKeyID := os.Getenv("PASSWORD_STORE_KEY"); gpgKeyID != "" {
		cfg.GPGKeyID = gpgKeyID
	}

	if cacheTimeoutStr := os.Getenv("PASSWORD_STORE_CACHE_TIMEOUT"); cacheTimeoutStr != "" {
		if timeout, err := strconv.Atoi(cacheTimeoutStr); err == nil && timeout >= 0 {
			cfg.CacheTimeout = timeout
		}
	}

	if gitURL := os.Getenv("PASSWORD_STORE_GIT_URL"); gitURL != "" {
		cfg.GitURL = gitURL
	}

	if gitAutoSyncStr := os.Getenv("PASSWORD_STORE_GIT_AUTO_SYNC"); gitAutoSyncStr != "" {
		if autoSync, err := strconv.ParseBool(gitAutoSyncStr); err == nil {
			cfg.GitAutoSync = autoSync
		}
	}

	// Load Git configuration from store directory if it exists
	cfg.loadGitConfig()

	return cfg, nil
}

// GitConfig represents the Git configuration stored in the password store
type GitConfig struct {
	URL      string `json:"url"`
	AutoSync bool   `json:"auto_sync"`
}

// loadGitConfig loads Git configuration from the store directory
func (cfg *Config) loadGitConfig() {
	gitConfigPath := filepath.Join(cfg.StoreDir, ".git-config")
	data, err := os.ReadFile(gitConfigPath)
	if err != nil {
		return // File doesn't exist or can't be read
	}

	var gitConfig GitConfig
	if err := json.Unmarshal(data, &gitConfig); err != nil {
		return // Invalid JSON
	}

	// Only use stored config if not overridden by environment variables
	if cfg.GitURL == "" {
		cfg.GitURL = gitConfig.URL
	}
	if os.Getenv("PASSWORD_STORE_GIT_AUTO_SYNC") == "" {
		cfg.GitAutoSync = gitConfig.AutoSync
	}
}

// SaveGitConfig saves Git configuration to the store directory
func (cfg *Config) SaveGitConfig() error {
	gitConfigPath := filepath.Join(cfg.StoreDir, ".git-config")

	gitConfig := GitConfig{
		URL:      cfg.GitURL,
		AutoSync: cfg.GitAutoSync,
	}

	data, err := json.MarshalIndent(gitConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(gitConfigPath, data, 0600)
}

func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
