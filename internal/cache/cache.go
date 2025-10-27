package cache

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a cached password entry stored on disk
type CacheEntry struct {
	EncryptedPassword []byte    `json:"encrypted_password"`
	Nonce             []byte    `json:"nonce"`
	Expiration        time.Time `json:"expiration"`
	SessionID         string    `json:"session_id"`
}

// PasswordCache manages cached master passwords with expiration
type PasswordCache struct {
	mu             sync.RWMutex
	cachedPassword string
	expiration     time.Time
	sessionID      string
	cacheTimeout   time.Duration
	cacheDir       string
}

// NewPasswordCache creates a new password cache instance
func NewPasswordCache(storeDir string, timeout time.Duration) *PasswordCache {
	cacheDir := filepath.Join(storeDir, ".cache")
	return &PasswordCache{
		cacheTimeout: timeout,
		cacheDir:     cacheDir,
	}
}

// Get retrieves the cached master password if valid and not expired
func (pc *PasswordCache) Get() (string, bool) {
	pc.mu.RLock()
	// First check in-memory cache
	if pc.cachedPassword != "" && time.Now().Before(pc.expiration) {
		defer pc.mu.RUnlock()
		return pc.cachedPassword, true
	}
	pc.mu.RUnlock()

	// Try to load from disk (need write lock to update cache)
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if password, loaded := pc.loadFromDisk(); loaded {
		// Update in-memory cache
		pc.cachedPassword = password
		return password, true
	}

	return "", false
}

// Set stores the master password in cache with expiration
func (pc *PasswordCache) Set(password string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Generate a new session ID
	sessionBytes := make([]byte, 16)
	if _, err := rand.Read(sessionBytes); err != nil {
		return err
	}
	pc.sessionID = hex.EncodeToString(sessionBytes)

	// Store password and set expiration
	pc.cachedPassword = password
	pc.expiration = time.Now().Add(pc.cacheTimeout)

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(pc.cacheDir, 0700); err != nil {
		return err
	}

	// Save to disk for persistence across processes
	if err := pc.saveToDisk(password); err != nil {
		return err
	}

	// Write session ID to file for process verification
	sessionFile := filepath.Join(pc.cacheDir, "session")
	return os.WriteFile(sessionFile, []byte(pc.sessionID), 0600)
}

// Clear removes the cached password
func (pc *PasswordCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.cachedPassword = ""
	pc.expiration = time.Time{}
	pc.sessionID = ""

	// Remove cache files
	sessionFile := filepath.Join(pc.cacheDir, "session")
	cacheFile := filepath.Join(pc.cacheDir, "password.cache")
	os.Remove(sessionFile)
	os.Remove(cacheFile)
}

// IsExpired checks if the cached password has expired
func (pc *PasswordCache) IsExpired() bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	return pc.cachedPassword == "" || time.Now().After(pc.expiration)
}

// GetRemainingTime returns the remaining time before expiration
func (pc *PasswordCache) GetRemainingTime() time.Duration {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if pc.cachedPassword == "" {
		return 0
	}

	remaining := time.Until(pc.expiration)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ValidateSession checks if the current session is still valid
func (pc *PasswordCache) ValidateSession() bool {
	pc.mu.RLock()
	sessionID := pc.sessionID
	pc.mu.RUnlock()

	if sessionID == "" {
		return false
	}

	sessionFile := filepath.Join(pc.cacheDir, "session")
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(sessionID), data) == 1
}

// SetTimeout updates the cache timeout duration
func (pc *PasswordCache) SetTimeout(timeout time.Duration) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.cacheTimeout = timeout
}

// GetTimeout returns the current cache timeout duration
func (pc *PasswordCache) GetTimeout() time.Duration {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.cacheTimeout
}

// generateCacheKey creates a key for encrypting the cached password
func (pc *PasswordCache) generateCacheKey() []byte {
	// Use a combination of session ID and a fixed string to derive encryption key
	h := sha256.New()
	h.Write([]byte(pc.sessionID))
	h.Write([]byte("chowkidaar-cache-key"))
	return h.Sum(nil)
}

// saveToDisk encrypts and saves the password to disk
func (pc *PasswordCache) saveToDisk(password string) error {
	// Generate encryption key from session ID
	key := pc.generateCacheKey()

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	// Encrypt password
	encryptedPassword := gcm.Seal(nil, nonce, []byte(password), nil)

	// Create cache entry
	entry := CacheEntry{
		EncryptedPassword: encryptedPassword,
		Nonce:             nonce,
		Expiration:        pc.expiration,
		SessionID:         pc.sessionID,
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Write to cache file
	cacheFile := filepath.Join(pc.cacheDir, "password.cache")
	return os.WriteFile(cacheFile, data, 0600)
}

// loadFromDisk loads and decrypts the password from disk
func (pc *PasswordCache) loadFromDisk() (string, bool) {
	cacheFile := filepath.Join(pc.cacheDir, "password.cache")

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return "", false
	}

	// Read cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return "", false
	}

	// Unmarshal JSON
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Invalid cache file, remove it
		os.Remove(cacheFile)
		return "", false
	}

	// Check expiration
	if time.Now().After(entry.Expiration) {
		// Expired, remove cache file
		os.Remove(cacheFile)
		return "", false
	}

	// Update session info
	pc.sessionID = entry.SessionID
	pc.expiration = entry.Expiration

	// Generate decryption key
	key := pc.generateCacheKey()

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		os.Remove(cacheFile)
		return "", false
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		os.Remove(cacheFile)
		return "", false
	}

	// Decrypt password
	password, err := gcm.Open(nil, entry.Nonce, entry.EncryptedPassword, nil)
	if err != nil {
		// Decryption failed, remove cache file
		os.Remove(cacheFile)
		return "", false
	}

	return string(password), true
}
