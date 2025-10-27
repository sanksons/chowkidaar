package cli

import (
	"fmt"

	"chowkidaar/internal/config"
	"chowkidaar/internal/list"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [subfolder]",
	Short: "List passwords",
	Long: `List names of passwords inside the tree at subfolder with a clean, modern view.
If no subfolder is provided, list all passwords.

The list command provides a beautiful tree view with icons and colors for easy navigation.`,
	Aliases: []string{"ls"},
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		subfolder := ""
		if len(args) > 0 {
			subfolder = args[0]
		}

		// Use the enhanced list view
		options := list.DefaultOptions()

		// Get flags
		if flat, _ := cmd.Flags().GetBool("flat"); flat {
			options.Flat = true
		}
		if details, _ := cmd.Flags().GetBool("details"); details {
			options.ShowDetails = true
		}
		if noIcons, _ := cmd.Flags().GetBool("no-icons"); noIcons {
			options.ShowIcons = false
		}
		if noColors, _ := cmd.Flags().GetBool("no-colors"); noColors {
			options.ShowColors = false
		}
		if maxDepth, _ := cmd.Flags().GetInt("max-depth"); maxDepth >= 0 {
			options.MaxDepth = maxDepth
		}
		if filter, _ := cmd.Flags().GetString("filter"); filter != "" {
			options.SearchFilter = filter
		}

		return list.GenerateWithOptions(cfg.StoreDir, subfolder, options)
	},
}

func init() {
	listCmd.Flags().BoolP("flat", "f", false, "Display as flat list instead of tree")
	listCmd.Flags().BoolP("details", "d", false, "Show details like modification date")
	listCmd.Flags().Bool("no-icons", false, "Disable emoji icons")
	listCmd.Flags().Bool("no-colors", false, "Disable color output")
	listCmd.Flags().Int("max-depth", -1, "Maximum depth to display (-1 for unlimited)")
	listCmd.Flags().String("filter", "", "Filter entries by name")
}
