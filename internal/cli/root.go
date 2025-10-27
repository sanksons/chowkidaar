package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "chowkidaar",
	Short: "Chowkidaar - Your faithful password guardian",
	Long: `Chowkidaar (चौकीदार) - A password manager that guards your secrets like a faithful watchman.
Stores encrypted passwords in a directory tree with Git synchronization support,
inspired by the Unix password store (pass). 

Features:
- Passwords encrypted using Argon2id + AES-256-GCM
- Hierarchical organization like file system
- Git sync for multi-device access
- Master password cached for 5 minutes by default

Use 'chowkidaar cache' commands to manage the cache behavior.`,
}

// Execute runs the CLI
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(insertCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(gitCmd)
}
