package cli

import (
	"fmt"

	"chowkidaar/internal/config"
	"chowkidaar/internal/store"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [subfolder]",
	Short: "List passwords",
	Long: `List names of passwords inside the tree at subfolder by using the tree program.
If no subfolder is provided, list all passwords.`,
	Aliases: []string{"ls"},
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

		subfolder := ""
		if len(args) > 0 {
			subfolder = args[0]
		}

		return passwordStore.List(subfolder)
	},
}
