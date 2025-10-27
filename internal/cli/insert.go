package cli

import (
	"fmt"

	"chowkidaar/internal/config"
	"chowkidaar/internal/store"

	"github.com/spf13/cobra"
)

var insertCmd = &cobra.Command{
	Use:   "insert [pass-name]",
	Short: "Insert a new password",
	Long: `Insert a new password into the password store.
The password name should be in the format of a file path (e.g., Email/gmail.com).

The master password will be cached for 5 minutes (configurable) to avoid repeated prompts.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		passName := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		passwordStore, err := store.NewWithGitConfig(cfg.StoreDir, cfg.CacheTimeout, cfg.GitURL, cfg.GitAutoSync)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}

		// Prompt for master password
		masterPassword, err := passwordStore.PromptMasterPassword("Enter master password: ")
		if err != nil {
			return fmt.Errorf("failed to read master password: %w", err)
		}

		// Prompt for password to store
		fmt.Printf("Enter password for %s: ", passName)
		var password string
		fmt.Scanln(&password)

		if err := passwordStore.Insert(passName, password, masterPassword); err != nil {
			return fmt.Errorf("failed to insert password: %w", err)
		}

		fmt.Printf("Password for '%s' inserted successfully\n", passName)
		return nil
	},
}

var multiline bool

func init() {
	insertCmd.Flags().BoolVarP(&multiline, "multiline", "m", false, "Enable multiline password entry")
}
