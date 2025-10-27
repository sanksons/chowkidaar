package gitsync

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/term"
)

// NetrcEntry represents a single entry in .netrc file
type NetrcEntry struct {
	Machine  string
	Login    string
	Password string
}

// GitSync handles Git operations for the password store
type GitSync struct {
	storeDir   string
	repository *gogit.Repository
	remoteURL  string
	auth       interface{} // Will hold either *http.BasicAuth or *ssh.PublicKeys
}

// NewGitSync creates a new GitSync instance
func NewGitSync(storeDir, remoteURL string) *GitSync {
	gs := &GitSync{
		storeDir:  storeDir,
		remoteURL: remoteURL,
	}

	// Try to open existing Git repository
	if repo, err := gogit.PlainOpen(storeDir); err == nil {
		gs.repository = repo
		// Ensure .gitignore is up to date
		gs.ensureGitignore()
	}

	return gs
}

// InitializeWithRemote initializes or clones a password store with Git support
func (gs *GitSync) InitializeWithRemote() error {
	// Check if the directory already exists and has content
	if _, err := os.Stat(gs.storeDir); err == nil {
		// Directory exists, check if it's already a Git repository
		if repo, err := gogit.PlainOpen(gs.storeDir); err == nil {
			gs.repository = repo
			return gs.configureRemote()
		}
		// Directory exists but not a Git repo, check if it's empty
		entries, err := os.ReadDir(gs.storeDir)
		if err != nil {
			return fmt.Errorf("failed to read store directory: %w", err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("store directory is not empty and not a Git repository")
		}
	}

	// Try to clone from remote
	if gs.remoteURL != "" {
		return gs.cloneFromRemote()
	}

	// Initialize local Git repository
	return gs.initLocalRepository()
}

// cloneFromRemote clones the password store from a remote Git repository
func (gs *GitSync) cloneFromRemote() error {
	fmt.Printf("Cloning password store from %s...\n", gs.remoteURL)

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(gs.storeDir), 0700); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Setup authentication if needed
	if err := gs.setupAuthentication(); err != nil {
		return fmt.Errorf("failed to setup authentication: %w", err)
	}

	// Clone the repository with authentication
	cloneOptions := &gogit.CloneOptions{
		URL:      gs.remoteURL,
		Progress: os.Stdout,
	}

	// Add authentication if available
	if gs.auth != nil {
		cloneOptions.Auth = gs.auth.(transport.AuthMethod)
	}

	repo, err := gogit.PlainClone(gs.storeDir, false, cloneOptions)

	if err != nil {
		// If clone fails, check if it's because the repo is empty
		if strings.Contains(err.Error(), "remote repository is empty") {
			fmt.Println("Remote repository is empty, initializing new password store...")
			return gs.initLocalRepository()
		}
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	gs.repository = repo
	fmt.Println("Password store cloned successfully!")

	// Ensure .gitignore is up to date after cloning
	if err := gs.ensureGitignore(); err != nil {
		fmt.Printf("Warning: failed to update .gitignore: %v\n", err)
	}

	// Count existing passwords
	count, err := gs.countPasswordFiles()
	if err == nil && count > 0 {
		fmt.Printf("Found %d existing passwords in the store.\n", count)
	}

	return nil
}

// initLocalRepository initializes a new local Git repository
func (gs *GitSync) initLocalRepository() error {
	fmt.Println("Initializing new password store with Git support...")

	// Create store directory
	if err := os.MkdirAll(gs.storeDir, 0700); err != nil {
		return fmt.Errorf("failed to create store directory: %w", err)
	}

	// Initialize Git repository
	repo, err := gogit.PlainInit(gs.storeDir, false)
	if err != nil {
		return fmt.Errorf("failed to initialize Git repository: %w", err)
	}

	gs.repository = repo

	// Configure remote if URL is provided
	if gs.remoteURL != "" {
		if err := gs.configureRemote(); err != nil {
			return err
		}
	}

	// Create initial .gitignore
	gitignorePath := filepath.Join(gs.storeDir, ".gitignore")
	gitignoreContent := `# Chowkidaar configuration and cache files
.cache/
.master
.git-config

# System files
.DS_Store
*.tmp
*.swp
*~

# Only backup encrypted password files (*.enc)
# Everything else should be ignored by default
`
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Create initial commit
	if err := gs.commitChanges("Initialize password store"); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	fmt.Println("Password store initialized successfully!")
	return nil
}

// configureRemote configures the remote origin for the repository
func (gs *GitSync) configureRemote() error {
	if gs.remoteURL == "" {
		return nil
	}

	// Check if remote already exists
	remotes, err := gs.repository.Remotes()
	if err != nil {
		return fmt.Errorf("failed to get remotes: %w", err)
	}

	// Remove existing origin if it exists
	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			gs.repository.DeleteRemote("origin")
			break
		}
	}

	// Add new origin
	_, err = gs.repository.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{gs.remoteURL},
	})

	if err != nil {
		return fmt.Errorf("failed to configure remote: %w", err)
	}

	return nil
}

// Push pushes changes to the remote repository
func (gs *GitSync) Push() error {
	if gs.repository == nil {
		return fmt.Errorf("Git repository not initialized")
	}

	fmt.Println("Pushing changes to remote repository...")

	// Setup authentication if not already done
	if gs.auth == nil {
		if err := gs.setupAuthentication(); err != nil {
			return fmt.Errorf("failed to setup authentication: %w", err)
		}
	}

	pushOptions := &gogit.PushOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	}

	// Add authentication if available
	if gs.auth != nil {
		pushOptions.Auth = gs.auth.(transport.AuthMethod)
	}

	err := gs.repository.Push(pushOptions)

	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	if err == gogit.NoErrAlreadyUpToDate {
		fmt.Println("Already up to date.")
	} else {
		fmt.Println("Changes pushed successfully!")
	}

	return nil
}

// Pull pulls changes from the remote repository
func (gs *GitSync) Pull() error {
	if gs.repository == nil {
		return fmt.Errorf("Git repository not initialized")
	}

	// Get the working tree
	worktree, err := gs.repository.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	fmt.Println("Pulling changes from remote repository...")

	// Setup authentication if not already done
	if gs.auth == nil {
		if err := gs.setupAuthentication(); err != nil {
			return fmt.Errorf("failed to setup authentication: %w", err)
		}
	}

	pullOptions := &gogit.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	}

	// Add authentication if available
	if gs.auth != nil {
		pullOptions.Auth = gs.auth.(transport.AuthMethod)
	}

	err = worktree.Pull(pullOptions)

	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	if err == gogit.NoErrAlreadyUpToDate {
		fmt.Println("Already up to date.")
	} else {
		fmt.Println("Changes pulled successfully!")
	}

	return nil
}

// CommitChanges commits changes to the repository
func (gs *GitSync) commitChanges(message string) error {
	if gs.repository == nil {
		return fmt.Errorf("Git repository not initialized")
	}

	// Get the working tree
	worktree, err := gs.repository.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Check if there are any changes to commit
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if len(status) == 0 {
		// No changes to commit
		return nil
	}

	// Commit changes
	commit, err := worktree.Commit(message, &gogit.CommitOptions{
		// Author: &object.Signature{
		// 	//Name:  "pwd-mngr",
		// 	//Email: "pwd-mngr@localhost",
		// 	When: time.Now(),
		// },
	})

	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	fmt.Printf("Changes committed: %s\n", commit.String()[:8])
	return nil
}

// CommitAndPushChanges commits changes and pushes them to remote
func (gs *GitSync) CommitAndPushChanges(message string) error {
	// Commit changes
	if err := gs.commitChanges(message); err != nil {
		return err
	}

	// Push to remote if configured
	if gs.remoteURL != "" {
		return gs.Push()
	}

	return nil
}

// Commit commits changes locally without pushing to remote
func (gs *GitSync) Commit(message string) error {
	return gs.commitChanges(message)
}

// Status returns the Git status of the repository
func (gs *GitSync) Status() (gogit.Status, error) {
	if gs.repository == nil {
		return nil, fmt.Errorf("Git repository not initialized")
	}

	worktree, err := gs.repository.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	return worktree.Status()
}

// IsGitEnabled checks if Git support is enabled for this store
func (gs *GitSync) IsGitEnabled() bool {
	return gs.repository != nil
}

// GetRemoteURL returns the configured remote URL
func (gs *GitSync) GetRemoteURL() string {
	return gs.remoteURL
}

// countPasswordFiles counts the number of .enc files in the store
func (gs *GitSync) countPasswordFiles() (int, error) {
	count := 0
	err := filepath.Walk(gs.storeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".enc") {
			count++
		}
		return nil
	})
	return count, err
}

// setupAuthentication sets up Git authentication based on the URL and available methods
func (gs *GitSync) setupAuthentication() error {
	if gs.remoteURL == "" {
		return nil
	}

	// Check if it's an SSH URL
	if strings.HasPrefix(gs.remoteURL, "git@") || strings.Contains(gs.remoteURL, "ssh://") {
		return gs.setupSSHAuthentication()
	}

	// For HTTPS URLs, try different authentication methods
	if strings.HasPrefix(gs.remoteURL, "https://") {
		return gs.setupHTTPSAuthentication()
	}

	return nil
}

// setupSSHAuthentication sets up SSH key authentication
func (gs *GitSync) setupSSHAuthentication() error {
	// Try to use SSH agent first
	sshAuth, err := ssh.NewSSHAgentAuth("git")
	if err == nil {
		gs.auth = sshAuth
		return nil
	}

	// Try to use default SSH key
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Try common SSH key locations
	keyPaths := []string{
		filepath.Join(homeDir, ".ssh", "id_rsa"),
		filepath.Join(homeDir, ".ssh", "id_ed25519"),
		filepath.Join(homeDir, ".ssh", "id_ecdsa"),
	}

	for _, keyPath := range keyPaths {
		if _, err := os.Stat(keyPath); err == nil {
			sshAuth, err := ssh.NewPublicKeysFromFile("git", keyPath, "")
			if err == nil {
				gs.auth = sshAuth
				return nil
			}
			// If key requires passphrase, prompt for it
			fmt.Printf("SSH key %s requires a passphrase: ", keyPath)
			passphrase, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				continue
			}
			sshAuth, err = ssh.NewPublicKeysFromFile("git", keyPath, string(passphrase))
			if err == nil {
				gs.auth = sshAuth
				return nil
			}
		}
	}

	return fmt.Errorf("no valid SSH authentication method found")
}

// setupHTTPSAuthentication sets up HTTPS authentication
func (gs *GitSync) setupHTTPSAuthentication() error {
	// First, try to read from .netrc file
	if username, password, err := gs.readNetrcCredentials(); err == nil && username != "" && password != "" {
		gs.auth = &http.BasicAuth{
			Username: username,
			Password: password,
		}
		fmt.Printf("Using credentials from .netrc file for authentication\n")
		return nil
	}

	// Check for Git credentials in environment variables
	if username := os.Getenv("GIT_USERNAME"); username != "" {
		password := os.Getenv("GIT_PASSWORD")
		if password == "" {
			password = os.Getenv("GIT_TOKEN") // Support both password and token
		}
		if password != "" {
			gs.auth = &http.BasicAuth{
				Username: username,
				Password: password,
			}
			return nil
		}
	}

	// Check if URL contains embedded credentials
	if strings.Contains(gs.remoteURL, "@") && !strings.HasPrefix(gs.remoteURL, "git@") {
		// URL already contains credentials, no additional auth needed
		return nil
	}

	// Prompt for credentials
	fmt.Print("Git username: ")
	var username string
	fmt.Scanln(&username)

	fmt.Print("Git password/token: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	gs.auth = &http.BasicAuth{
		Username: username,
		Password: string(password),
	}

	return nil
}

// readNetrcCredentials reads credentials from .netrc file for the current remote URL
func (gs *GitSync) readNetrcCredentials() (string, string, error) {
	if gs.remoteURL == "" {
		return "", "", fmt.Errorf("no remote URL configured")
	}

	// Parse the remote URL to get the hostname
	parsedURL, err := url.Parse(gs.remoteURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse remote URL: %w", err)
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return "", "", fmt.Errorf("could not extract hostname from URL: %s", gs.remoteURL)
	}

	// Try to read .netrc file
	netrcEntries, err := gs.parseNetrcFile()
	if err != nil {
		return "", "", err
	}

	// Look for matching entry
	for _, entry := range netrcEntries {
		if entry.Machine == hostname || entry.Machine == "default" {
			return entry.Login, entry.Password, nil
		}
	}

	return "", "", fmt.Errorf("no matching entry found in .netrc for %s", hostname)
}

// parseNetrcFile parses the .netrc file and returns all entries
func (gs *GitSync) parseNetrcFile() ([]NetrcEntry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Try both .netrc and _netrc (Windows)
	netrcPaths := []string{
		filepath.Join(homeDir, ".netrc"),
		filepath.Join(homeDir, "_netrc"),
	}

	var netrcPath string
	for _, path := range netrcPaths {
		if _, err := os.Stat(path); err == nil {
			netrcPath = path
			break
		}
	}

	if netrcPath == "" {
		return nil, fmt.Errorf(".netrc file not found")
	}

	file, err := os.Open(netrcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .netrc file: %w", err)
	}
	defer file.Close()

	var entries []NetrcEntry
	var currentEntry NetrcEntry

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)

		for i := 0; i < len(fields); i += 2 {
			if i+1 >= len(fields) {
				break
			}

			key := fields[i]
			value := fields[i+1]

			switch key {
			case "machine":
				// Save previous entry if exists
				if currentEntry.Machine != "" {
					entries = append(entries, currentEntry)
				}
				// Start new entry
				currentEntry = NetrcEntry{Machine: value}
			case "default":
				// Save previous entry if exists
				if currentEntry.Machine != "" {
					entries = append(entries, currentEntry)
				}
				// Start new default entry
				currentEntry = NetrcEntry{Machine: "default"}
			case "login":
				currentEntry.Login = value
			case "password":
				currentEntry.Password = value
			}
		}
	}

	// Add the last entry
	if currentEntry.Machine != "" {
		entries = append(entries, currentEntry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .netrc file: %w", err)
	}

	return entries, nil
}

// ensureGitignore creates or updates the .gitignore file to exclude config files
func (gs *GitSync) ensureGitignore() error {
	if gs.repository == nil {
		return nil // No Git repository, nothing to do
	}

	gitignorePath := filepath.Join(gs.storeDir, ".gitignore")
	gitignoreContent := `# Chowkidaar configuration and cache files
.cache/
.master
.git-config

# System files
.DS_Store
*.tmp
*.swp
*~

# Only backup encrypted password files (*.enc)
# Everything else should be ignored by default
`

	// Write the .gitignore file
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return err
	}

	// Remove config files from Git tracking if they were previously committed
	return gs.removeTrackedConfigFiles()
}

// removeTrackedConfigFiles removes config files from Git tracking if they were previously committed
func (gs *GitSync) removeTrackedConfigFiles() error {
	if gs.repository == nil {
		return nil
	}

	worktree, err := gs.repository.Worktree()
	if err != nil {
		return err
	}

	configFiles := []string{".cache", ".master", ".git-config"}

	for _, configFile := range configFiles {
		configPath := filepath.Join(gs.storeDir, configFile)

		// Check if file exists and is tracked by Git
		if _, err := os.Stat(configPath); err == nil {
			// Try to remove from Git tracking (ignore errors if file is not tracked)
			worktree.Remove(configFile)
		}
	}

	return nil
}
