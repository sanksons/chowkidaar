package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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

For existing stores (with .enc files), you'll need to enter the 12-word recovery phrase.
For new stores, a recovery phrase will be generated and displayed.

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
		} else {
			// Create password store directory for local-only initialization
			if err := os.MkdirAll(storeDir, 0700); err != nil {
				return fmt.Errorf("failed to create store directory: %w", err)
			}
		}

		// Initialize crypto handler
		cryptoHandler := crypto.New(storeDir)

		// Check if this is an existing store (has encrypted passwords)
		hasEncryptedPasswords, err := cryptoHandler.HasEncryptedPasswords()
		if err != nil {
			return fmt.Errorf("failed to check for encrypted passwords: %w", err)
		}

		if hasEncryptedPasswords {
			// SCENARIO: Cloning existing password store
			fmt.Println("\n🔐 Existing password store detected!")
			fmt.Println("To access these passwords, you need the 12-word recovery phrase.")
			fmt.Println()

			// Prompt for mnemonic
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter your 12-word recovery phrase: ")
			mnemonicInput, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read recovery phrase: %w", err)
			}
			mnemonic := strings.TrimSpace(mnemonicInput)

			// Create keyfile from mnemonic
			if err := cryptoHandler.CreateKeyFileFromMnemonic(mnemonic); err != nil {
				return fmt.Errorf("failed to create keyfile from recovery phrase: %w", err)
			}

			// Save Git configuration
			if gitURL != "" {
				cfg.GitURL = gitURL
				if err := cfg.SaveGitConfig(); err != nil {
					fmt.Printf("Warning: failed to save Git configuration: %v\n", err)
				}
			}

			fmt.Printf("\n✅ Password store initialized successfully!\n")
			fmt.Printf("Store location: %s\n", storeDir)
			if gitURL != "" {
				fmt.Printf("Git remote: %s\n", gitURL)
			}
			fmt.Printf("\nThe store is ready to use. You can:\n")
			fmt.Printf("- View passwords: chowkidaar list\n")
			fmt.Printf("- Show a password: chowkidaar show <name>\n")
			fmt.Printf("- Add new passwords: chowkidaar insert <name>\n")
			if gitURL != "" {
				fmt.Printf("- Sync changes: chowkidaar git push/pull\n")
			}
			return nil
		}

		// SCENARIO: Creating new password store
		
		// Check if keyfile already exists (store was previously initialized)
		if cryptoHandler.HasKeyFile() {
			return fmt.Errorf("password store already initialized at %s", storeDir)
		}

		fmt.Println("\n🆕 Creating new password store...")
		
		// Generate BIP-39 mnemonic
		mnemonic, err := cryptoHandler.GenerateMnemonic()
		if err != nil {
			return fmt.Errorf("failed to generate recovery phrase: %w", err)
		}

		// Create keyfile from mnemonic
		if err := cryptoHandler.CreateKeyFileFromMnemonic(mnemonic); err != nil {
			return fmt.Errorf("failed to create keyfile: %w", err)
		}

		// Prompt for master password
		fmt.Println("\nSetting up master password for the password store...")
		masterPassword, err := promptPasswordInput("Enter master password: ")
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

		// Save Git configuration if Git URL was provided
		if gitURL != "" {
			cfg.GitURL = gitURL
			if err := cfg.SaveGitConfig(); err != nil {
				fmt.Printf("Warning: failed to save Git configuration: %v\n", err)
			}
		}

		// Display success message with recovery phrase
		fmt.Printf("\n✅ Password store initialized successfully!\n")
		fmt.Printf("Store location: %s\n", storeDir)
		if gitURL != "" {
			fmt.Printf("Git remote: %s\n", gitURL)
		}
		
		fmt.Println("\n" + strings.Repeat("=", 70))
		fmt.Println("⚠️  IMPORTANT: Write down your 12-word recovery phrase!")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("\n%s\n\n", mnemonic)
		fmt.Println("This phrase is required to:")
		fmt.Println("  • Set up this password store on another device")
		fmt.Println("  • Recover access if you lose your keyfile")
		fmt.Println("\n⚠️  Store this phrase safely - it CANNOT be recovered if lost!")
		fmt.Println(strings.Repeat("=", 70))

		fmt.Printf("\nYou can now:\n")
		fmt.Printf("- Add passwords: chowkidaar insert <name>\n")
		fmt.Printf("- View passwords: chowkidaar list\n")
		if gitURL != "" {
			fmt.Printf("- Sync changes: chowkidaar git push/pull\n")
		}

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&gitURL, "git-url", "", "Git repository URL to clone existing passwords or sync new ones")
}
