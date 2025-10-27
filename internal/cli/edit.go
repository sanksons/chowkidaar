package cli

import (
	"fmt"

	"chowkidaar/internal/config"
	"chowkidaar/internal/store"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [pass-name]",
	Short: "Edit existing password",
	Long: `Insert a new password or edit an existing password using your default editor.
The password will be encrypted and stored in the password store.

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

		if err := passwordStore.Edit(passName, masterPassword, cfg.Editor); err != nil {
			return fmt.Errorf("failed to edit password: %w", err)
		}

		fmt.Printf("Password for '%s' updated successfully\n", passName)
		return nil
	},
}
