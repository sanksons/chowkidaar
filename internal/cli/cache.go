package cli

import (
	"fmt"
	"time"

	"chowkidaar/internal/config"
	"chowkidaar/internal/store"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage master password cache",
	Long: `Manage the master password cache. This command allows you to:
- Check cache status and remaining time
- Clear the cached master password
- Configure cache timeout`,
}

var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache status",
	Long:  `Display the current status of the master password cache, including remaining time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		passwordStore, err := store.NewWithGitConfig(cfg.StoreDir, cfg.CacheTimeout, cfg.GitURL, cfg.GitAutoSync)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}

		isValid, remaining := passwordStore.GetCacheStatus()

		if isValid {
			minutes := int(remaining.Minutes())
			seconds := int(remaining.Seconds()) % 60
			fmt.Printf("Master password is cached for %d minutes and %d seconds\n", minutes, seconds)
		} else {
			fmt.Println("No master password cached")
		}

		fmt.Printf("Cache timeout configured for: %d minutes\n", cfg.CacheTimeout)
		return nil
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear cached master password",
	Long:  `Clear the cached master password, forcing re-authentication on the next operation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		passwordStore, err := store.NewWithGitConfig(cfg.StoreDir, cfg.CacheTimeout, cfg.GitURL, cfg.GitAutoSync)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}

		passwordStore.ClearPasswordCache()
		fmt.Println("Master password cache cleared")
		return nil
	},
}

var cacheTimeoutCmd = &cobra.Command{
	Use:   "timeout [minutes]",
	Short: "Set cache timeout duration",
	Long: `Set the cache timeout duration in minutes for the current session.
This will only affect the current session and won't change the default configuration.
To permanently change the timeout, set the PASSWORD_STORE_CACHE_TIMEOUT environment variable.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var minutes int
		if _, err := fmt.Sscanf(args[0], "%d", &minutes); err != nil || minutes < 0 {
			return fmt.Errorf("invalid timeout value. Please provide a positive number of minutes")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		passwordStore, err := store.NewWithGitConfig(cfg.StoreDir, cfg.CacheTimeout, cfg.GitURL, cfg.GitAutoSync)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}

		timeout := time.Duration(minutes) * time.Minute
		passwordStore.SetCacheTimeout(timeout)

		if minutes == 0 {
			fmt.Println("Cache timeout set to 0 minutes (caching disabled)")
		} else {
			fmt.Printf("Cache timeout set to %d minutes for this session\n", minutes)
		}

		return nil
	},
}

func init() {
	// Add subcommands to cache command
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheTimeoutCmd)
}
