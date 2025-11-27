package store

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"chowkidaar/internal/crypto"
	"chowkidaar/internal/gitsync"
)

const (
	defaultCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	symbolCharset  = "!@#$%^&*()_+-=[]{}|;:,.<>?"
)

// Store represents a password store
type Store struct {
	baseDir  string
	crypto   *crypto.Crypto
	gitSync  *gitsync.GitSync
	autoSync bool
}

// New creates a new password store instance
func New(baseDir string) (*Store, error) {
	return NewWithConfig(baseDir, 5) // Default 5 minutes timeout
}

// NewWithConfig creates a new password store instance with custom cache timeout
func NewWithConfig(baseDir string, cacheTimeoutMinutes int) (*Store, error) {
	return NewWithGitConfig(baseDir, cacheTimeoutMinutes, "", false)
}

// NewWithGitConfig creates a new password store instance with Git configuration
func NewWithGitConfig(baseDir string, cacheTimeoutMinutes int, gitURL string, autoSync bool) (*Store, error) {
	// Check if store directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("password store not initialized. Run 'chowkidaar init' first")
	}

	cryptoHandler, err := crypto.NewFromStore(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize crypto: %w", err)
	}

	// Set the cache timeout
	cryptoHandler.SetCacheTimeout(time.Duration(cacheTimeoutMinutes) * time.Minute)

	// Initialize Git sync if URL is provided
	var gitSync *gitsync.GitSync
	if gitURL != "" {
		gitSync = gitsync.NewGitSync(baseDir, gitURL)
	}

	return &Store{
		baseDir:  baseDir,
		crypto:   cryptoHandler,
		gitSync:  gitSync,
		autoSync: autoSync,
	}, nil
}

// PromptMasterPassword prompts for the master password
func (s *Store) PromptMasterPassword(prompt string) (string, error) {
	return s.crypto.PromptMasterPassword(prompt)
}

// Insert stores a new password
func (s *Store) Insert(name, password, masterPassword string) error {
	// Validate password against existing encrypted files (if any)
	if err := s.validatePasswordIfNeeded(masterPassword); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	// Check if password already exists
	if s.Exists(name) {
		return fmt.Errorf("password '%s' already exists", name)
	}

	filePath := s.getPasswordFilePath(name)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Encrypt password
	encrypted, err := s.crypto.Encrypt([]byte(password), masterPassword)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Write encrypted password to file
	if err := os.WriteFile(filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write password file: %w", err)
	}

	// Cache the validated master password (encryption succeeded)
	s.crypto.CachePassword(masterPassword)

	// Auto-commit to Git if enabled
	if err := s.autoCommit(fmt.Sprintf("Add password for %s", name)); err != nil {
		fmt.Printf("Warning: failed to commit changes to Git: %v\n", err)
	}

	return nil
}

// Update updates an existing password or creates a new one if it doesn't exist
func (s *Store) Update(name, password, masterPassword string) error {
	// Validate password against existing encrypted files (if any)
	if err := s.validatePasswordIfNeeded(masterPassword); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	filePath := s.getPasswordFilePath(name)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Encrypt password
	encrypted, err := s.crypto.Encrypt([]byte(password), masterPassword)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	// Write encrypted password to file (overwrite if exists)
	if err := os.WriteFile(filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write password file: %w", err)
	}

	// Cache the validated master password (encryption succeeded)
	s.crypto.CachePassword(masterPassword)

	// Auto-commit to Git if enabled
	if err := s.autoCommit(fmt.Sprintf("Update password for %s", name)); err != nil {
		fmt.Printf("Warning: failed to commit changes to Git: %v\n", err)
	}

	return nil
}

// Show retrieves and decrypts a password
func (s *Store) Show(name, masterPassword string) (string, error) {
	filePath := s.getPasswordFilePath(name)

	// Read encrypted password
	encrypted, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("password '%s' does not exist", name)
		}
		return "", fmt.Errorf("failed to read password file: %w", err)
	}

	// Decrypt password
	decrypted, err := s.crypto.Decrypt(encrypted, masterPassword)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Cache the validated master password (decryption succeeded)
	s.crypto.CachePassword(masterPassword)

	return string(decrypted), nil
}

// Generate creates and stores a new random password
func (s *Store) Generate(name string, length int, noSymbols bool, inPlace bool, masterPassword string) (string, error) {
	charset := defaultCharset
	if !noSymbols {
		charset += symbolCharset
	}

	password, err := generatePassword(length, charset)
	if err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}

	if err := s.Insert(name, password, masterPassword); err != nil {
		return "", fmt.Errorf("failed to insert generated password: %w", err)
	}

	// Note: Password is already cached in Insert() method

	if inPlace {
		return "", nil
	}

	return password, nil
}

// List displays the password store tree
func (s *Store) List(subfolder string) error {
	searchDir := s.baseDir
	if subfolder != "" {
		searchDir = filepath.Join(s.baseDir, subfolder)
	}

	// Check if tree command is available
	if _, err := exec.LookPath("tree"); err == nil {
		// Use tree command if available
		args := []string{"-C", "-l", "--noreport", searchDir}
		cmd := exec.Command("tree", args...)
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}

	// Fallback to custom tree implementation
	return s.listDirectory(searchDir, "")
}

// Exists checks if a password exists
func (s *Store) Exists(name string) bool {
	filePath := s.getPasswordFilePath(name)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// Remove deletes a password
func (s *Store) Remove(name string) error {
	filePath := s.getPasswordFilePath(name)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("password '%s' does not exist", name)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove password file: %w", err)
	}

	// Remove empty directories
	s.cleanupEmptyDirs(filepath.Dir(filePath))

	// Auto-commit to Git if enabled
	if err := s.autoCommit(fmt.Sprintf("Remove password for %s", name)); err != nil {
		fmt.Printf("Warning: failed to commit changes to Git: %v\n", err)
	}

	return nil
}

// ClearPasswordCache clears the cached master password
func (s *Store) ClearPasswordCache() {
	s.crypto.ClearPasswordCache()
}

// SetCacheTimeout sets the cache timeout duration
func (s *Store) SetCacheTimeout(timeout time.Duration) {
	s.crypto.SetCacheTimeout(timeout)
}

// GetCacheStatus returns information about the password cache
func (s *Store) GetCacheStatus() (bool, time.Duration) {
	isValid := s.crypto.IsCacheValid()
	remaining := s.crypto.GetCacheRemainingTime()
	return isValid, remaining
}

// Edit opens a password for editing using the specified editor
func (s *Store) Edit(name, masterPassword, editor string) error {
	filePath := s.getPasswordFilePath(name)

	// Check if password exists, if not create a new one
	var currentContent string
	if _, err := os.Stat(filePath); err == nil {
		// File exists, decrypt current content
		decrypted, err := s.Show(name, masterPassword)
		if err != nil {
			return fmt.Errorf("failed to read existing password: %w", err)
		}
		currentContent = decrypted
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check password file: %w", err)
	}
	// If file doesn't exist, currentContent remains empty string

	// Create temporary file for editing
	tmpFile, err := os.CreateTemp("", "chowkidaar-edit-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure temporary file is removed after editing
	defer func() {
		os.Remove(tmpPath)
	}()

	// Write current content to temporary file
	if _, err := tmpFile.WriteString(currentContent); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tmpFile.Close()

	// Open editor
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	// Read the edited content
	editedContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read edited content: %w", err)
	}

	// Remove trailing newline if it exists (common with editors)
	newPassword := strings.TrimSuffix(string(editedContent), "\n")

	// Check if content was changed
	if newPassword == currentContent {
		fmt.Printf("No changes made to '%s'\n", name)
		return nil
	}

	// Save the new password (use Update to allow overwriting existing passwords)
	if err := s.Update(name, newPassword, masterPassword); err != nil {
		return fmt.Errorf("failed to save edited password: %w", err)
	}

	return nil
}

// Helper methods

// validatePasswordIfNeeded validates the master password against an existing encrypted file
// This ensures password consistency across all operations
func (s *Store) validatePasswordIfNeeded(masterPassword string) error {
	// Find any .enc file to validate against
	var testFile string
	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".enc") {
			testFile = path
			return filepath.SkipAll // Stop after finding first .enc file
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to search for encrypted files: %w", err)
	}

	// If no encrypted files exist yet, password is valid (first time use)
	if testFile == "" {
		return nil
	}

	// Try to decrypt the test file to validate the password
	encrypted, err := os.ReadFile(testFile)
	if err != nil {
		return fmt.Errorf("failed to read test file: %w", err)
	}

	_, err = s.crypto.Decrypt(encrypted, masterPassword)
	if err != nil {
		return fmt.Errorf("incorrect master password")
	}

	// Password is valid, cache it
	s.crypto.CachePassword(masterPassword)

	return nil
}

func (s *Store) getPasswordFilePath(name string) string {
	// Ensure the name ends with .enc extension
	if !strings.HasSuffix(name, ".enc") {
		name += ".enc"
	}
	return filepath.Join(s.baseDir, name)
}

func (s *Store) listDirectory(dir, prefix string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for i, entry := range entries {
		// Skip hidden files starting with .
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		isLast := i == len(entries)-1

		var symbol string
		if isLast {
			symbol = "└── "
		} else {
			symbol = "├── "
		}

		name := strings.TrimSuffix(entry.Name(), ".enc")

		fmt.Printf("%s%s%s\n", prefix, symbol, name)

		if entry.IsDir() {
			nextPrefix := prefix
			if !isLast {
				nextPrefix += "│   "
			} else {
				nextPrefix += "    "
			}
			s.listDirectory(filepath.Join(dir, entry.Name()), nextPrefix)
		}
	}

	return nil
}

func (s *Store) cleanupEmptyDirs(dir string) {
	// Don't remove the base directory
	if dir == s.baseDir {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// If directory is empty, remove it and check parent
	if len(entries) == 0 {
		os.Remove(dir)
		s.cleanupEmptyDirs(filepath.Dir(dir))
	}
}

func generatePassword(length int, charset string) (string, error) {
	password := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		password[i] = charset[randomIndex.Int64()]
	}

	return string(password), nil
}

// autoCommit commits changes to Git if auto-sync is enabled
func (s *Store) autoCommit(message string) error {
	if s.gitSync == nil || !s.autoSync {
		return nil
	}

	return s.gitSync.Commit(message)
}
