package cli

import (
	"fmt"
	"os"
	"syscall"

	"chowkidaar/internal/config"
	"chowkidaar/internal/crypto"
	"chowkidaar/internal/gitsync"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var gitURL string

// promptPasswordInput prompts the user for a password without echoing it to the terminal
func promptPasswordInput(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after password input
	if err != nil {
		return "", err
	}
	return string(password), nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize new password store",
	Long: `Initialize a new password store directory with optional Git synchronization.
You will be prompted to set a master password that will be used to encrypt all stored passwords.

If --git-url is provided, the command will:
- Clone an existing password store from the remote repository if it exists
- Initialize a new Git repository and link it to the remote if the repository is empty
- Sync existing passwords from the remote repository

Examples:
  chowkidaar init                                    # Initialize local store only
  chowkidaar init --git-url https://github.com/user/passwords.git  # Clone or init with Git sync`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		storeDir := cfg.StoreDir

		// Initialize Git sync if URL is provided
		var gitSync *gitsync.GitSync
		if gitURL != "" {
			gitSync = gitsync.NewGitSync(storeDir, gitURL)

			// Initialize or clone the repository
			if err := gitSync.InitializeWithRemote(); err != nil {
				return fmt.Errorf("failed to initialize Git repository: %w", err)
			}

			// If we cloned an existing repository, check if it's already initialized
			cryptoHandler := crypto.New(storeDir)
			if cryptoHandler.IsStoreInitialized() {
				// Save Git configuration for existing store
				cfg.GitURL = gitURL
				if err := cfg.SaveGitConfig(); err != nil {
					fmt.Printf("Warning: failed to save Git configuration: %v\n", err)
				}

				fmt.Printf("Password store with existing passwords cloned successfully!\n")
				fmt.Printf("Store location: %s\n", storeDir)
				fmt.Printf("Git remote: %s\n", gitURL)
				fmt.Printf("\nThe store is ready to use. You can:\n")
				fmt.Printf("- View passwords: chowkidaar list\n")
				fmt.Printf("- Show a password: chowkidaar show <name>\n")
				fmt.Printf("- Add new passwords: chowkidaar insert <name>\n")
				fmt.Printf("- Sync changes: chowkidaar git push/pull\n")
				return nil
			}
		} else {
			// Create password store directory for local-only initialization
			if err := os.MkdirAll(storeDir, 0700); err != nil {
				return fmt.Errorf("failed to create store directory: %w", err)
			}
		}

		// Initialize crypto handler
		cryptoHandler := crypto.New(storeDir)

		// Check if already initialized (for local repositories)
		if cryptoHandler.IsStoreInitialized() {
			return fmt.Errorf("password store already initialized at %s", storeDir)
		}

		// Prompt for master password (without validation since we're initializing)
		fmt.Println("\nSetting up master password for the password store...")
		masterPassword, err := promptPasswordInput("Enter master password for this store: ")
		if err != nil {
			return fmt.Errorf("failed to read master password: %w", err)
		}

		if len(masterPassword) == 0 {
			return fmt.Errorf("master password cannot be empty")
		}

		// Confirm master password
		confirmPassword, err := promptPasswordInput("Confirm master password: ")
		if err != nil {
			return fmt.Errorf("failed to read password confirmation: %w", err)
		}

		if masterPassword != confirmPassword {
			return fmt.Errorf("passwords do not match")
		}

		// Initialize master password
		if err := cryptoHandler.InitializeMasterPassword(masterPassword); err != nil {
			return fmt.Errorf("failed to initialize master password: %w", err)
		}

		// Save Git configuration if Git URL was provided
		if gitURL != "" {
			cfg.GitURL = gitURL
			if err := cfg.SaveGitConfig(); err != nil {
				fmt.Printf("Warning: failed to save Git configuration: %v\n", err)
			}
		}

		// If Git sync is enabled, commit the master password file
		if gitSync != nil && gitSync.IsGitEnabled() {
			if err := gitSync.CommitAndPushChanges("Add master password configuration"); err != nil {
				fmt.Printf("Warning: failed to commit master password to Git: %v\n", err)
			}
		}

		fmt.Printf("\nPassword store initialized successfully!\n")
		fmt.Printf("Store location: %s\n", storeDir)
		if gitURL != "" {
			fmt.Printf("Git remote: %s\n", gitURL)
			fmt.Printf("\nGit synchronization is enabled. Changes will be automatically committed and pushed.\n")
		}
		fmt.Printf("\nYou can now:\n")
		fmt.Printf("- Add passwords: chowkidaar insert <name>\n")
		fmt.Printf("- View passwords: chowkidaar list\n")
		if gitURL != "" {
			fmt.Printf("- Sync changes: chowkidaar git push/pull\n")
		}
		fmt.Printf("\nRemember your master password - it cannot be recovered if lost!\n")

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&gitURL, "git-url", "", "Git repository URL to clone existing passwords or sync new ones")
}
