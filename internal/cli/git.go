package cli

import (
	"fmt"
	"strings"

	"chowkidaar/internal/config"
	"chowkidaar/internal/gitsync"

	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git synchronization commands",
	Long: `Manage Git synchronization for the password store.
These commands allow you to sync your password store with a remote Git repository,
similar to the Unix 'pass' password manager.

Available commands:
  status  - Show Git repository status
  push    - Push changes to remote repository  
  pull    - Pull changes from remote repository
  sync    - Pull then push (full synchronization)`,
}

var gitStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Git repository status",
	Long:  `Display the current Git status of the password store, showing modified, added, and deleted files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		gitSync := gitsync.NewGitSync(cfg.StoreDir, cfg.GitURL)

		if !gitSync.IsGitEnabled() {
			return fmt.Errorf("Git is not initialized for this password store. Run 'chowkidaar init --git-url <url>' to enable Git sync")
		}

		status, err := gitSync.Status()
		if err != nil {
			return fmt.Errorf("failed to get Git status: %w", err)
		}

		if len(status) == 0 {
			fmt.Println("Working tree clean - no changes to commit")
			return nil
		}

		fmt.Println("Changes in password store:")
		for file, fileStatus := range status {
			var statusStr string
			switch {
			case fileStatus.Staging != 0:
				statusStr = "staged"
			case fileStatus.Worktree != 0:
				statusStr = "modified"
			default:
				statusStr = "unknown"
			}

			// Remove .enc extension for cleaner output
			displayName := strings.TrimSuffix(file, ".enc")
			fmt.Printf("  %s: %s\n", statusStr, displayName)
		}

		fmt.Printf("\nRemote repository: %s\n", gitSync.GetRemoteURL())
		return nil
	},
}

var gitPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push changes to remote repository",
	Long:  `Commit any local changes and push them to the remote Git repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		gitSync := gitsync.NewGitSync(cfg.StoreDir, cfg.GitURL)

		if !gitSync.IsGitEnabled() {
			return fmt.Errorf("Git is not initialized for this password store. Run 'chowkidaar init --git-url <url>' to enable Git sync")
		}

		// Check if there are any changes to commit
		status, err := gitSync.Status()
		if err != nil {
			return fmt.Errorf("failed to get Git status: %w", err)
		}

		if len(status) > 0 {
			// Commit changes with a generic message
			if err := gitSync.CommitAndPushChanges("Update password store"); err != nil {
				return fmt.Errorf("failed to commit and push changes: %w", err)
			}
		} else {
			// Just push if no local changes
			if err := gitSync.Push(); err != nil {
				return fmt.Errorf("failed to push changes: %w", err)
			}
		}

		return nil
	},
}

var gitPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull changes from remote repository",
	Long:  `Pull and merge changes from the remote Git repository into the local password store.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		gitSync := gitsync.NewGitSync(cfg.StoreDir, cfg.GitURL)

		if !gitSync.IsGitEnabled() {
			return fmt.Errorf("Git is not initialized for this password store. Run 'chowkidaar init --git-url <url>' to enable Git sync")
		}

		if err := gitSync.Pull(); err != nil {
			return fmt.Errorf("failed to pull changes: %w", err)
		}

		return nil
	},
}

var gitSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Full synchronization (pull then push)",
	Long: `Perform a full synchronization with the remote repository:
1. Pull changes from remote
2. Commit any local changes  
3. Push committed changes to remote

This ensures your local store is up-to-date and your changes are backed up.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		gitSync := gitsync.NewGitSync(cfg.StoreDir, cfg.GitURL)

		if !gitSync.IsGitEnabled() {
			return fmt.Errorf("Git is not initialized for this password store. Run 'chowkidaar init --git-url <url>' to enable Git sync")
		}

		// Step 1: Pull changes from remote
		fmt.Println("Step 1: Pulling changes from remote...")
		if err := gitSync.Pull(); err != nil {
			return fmt.Errorf("failed to pull changes: %w", err)
		}

		// Step 2: Check for local changes and commit/push if any
		fmt.Println("Step 2: Checking for local changes...")
		status, err := gitSync.Status()
		if err != nil {
			return fmt.Errorf("failed to get Git status: %w", err)
		}

		if len(status) > 0 {
			fmt.Println("Step 3: Committing and pushing local changes...")
			if err := gitSync.CommitAndPushChanges("Sync password store"); err != nil {
				return fmt.Errorf("failed to commit and push changes: %w", err)
			}
		} else {
			fmt.Println("Step 3: No local changes to push.")
		}

		fmt.Println("Synchronization completed successfully!")
		return nil
	},
}

func init() {
	// Add subcommands to git command
	gitCmd.AddCommand(gitStatusCmd)
	gitCmd.AddCommand(gitPushCmd)
	gitCmd.AddCommand(gitPullCmd)
	gitCmd.AddCommand(gitSyncCmd)
}
