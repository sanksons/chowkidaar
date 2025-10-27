package cli

import (
	"fmt"

	"chowkidaar/internal/config"
	"chowkidaar/internal/store"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [pass-name]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove existing password",
	Long: `Remove the password named pass-name from the password store.
This command will prompt for confirmation before removing the password.`,
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

		if !force {
			fmt.Printf("Are you sure you want to delete '%s'? [y/N]: ", passName)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" && response != "yes" {
				fmt.Println("Password removal cancelled.")
				return nil
			}
		}

		if err := passwordStore.Remove(passName); err != nil {
			return fmt.Errorf("failed to remove password: %w", err)
		}

		fmt.Printf("Password '%s' removed successfully\n", passName)
		return nil
	},
}

var force bool

func init() {
	removeCmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal without confirmation")
}
