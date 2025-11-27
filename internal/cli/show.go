package cli

import (
	"fmt"

	"chowkidaar/internal/config"
	"chowkidaar/internal/store"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [pass-name]",
	Short: "Show existing password",
	Long: `Decrypt and print a password to stdout.
If no password name is provided, list all passwords.

The master password will be cached for 5 minutes (configurable) to avoid repeated prompts.`,
	Aliases: []string{"view", "get"},
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		passwordStore, err := store.NewWithGitConfig(cfg.StoreDir, cfg.CacheTimeout, cfg.GitURL, cfg.GitAutoSync)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}

		if len(args) == 0 {
			// List all passwords
			return passwordStore.List("")
		}

		passName := args[0]

		// Prompt for master password
		masterPassword, err := passwordStore.PromptMasterPassword("Enter master password: ")
		if err != nil {
			return fmt.Errorf("failed to read master password: %w", err)
		}

		password, err := passwordStore.Show(passName, masterPassword)
		if err != nil {
			return fmt.Errorf("failed to retrieve password: %w", err)
		}

		fmt.Print(password)
		return nil
	},
}

var clipboardFlag bool

func init() {
	showCmd.Flags().BoolVarP(&clipboardFlag, "clip", "c", false, "Copy password to clipboard")
}
