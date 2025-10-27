package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"chowkidaar/internal/cache"

	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

const (
	// Argon2 parameters (following OWASP recommendations)
	argon2Time    = 3         // Number of iterations
	argon2Memory  = 64 * 1024 // Memory in KB (64 MB)
	argon2Threads = 4         // Number of parallel threads
	argon2KeyLen  = 32        // Length of derived key (256 bits)

	// Salt and nonce sizes
	saltSize  = 32 // 256 bits
	nonceSize = 12 // 96 bits for GCM

	// Master password validation
	masterPasswordFile = ".master"
	masterSaltSize     = 32 // Salt for master password hash
)

// EncryptedData represents the structure of encrypted password data
type EncryptedData struct {
	Salt       []byte
	Nonce      []byte
	Ciphertext []byte
}

// Crypto handles password-based encryption using Argon2id + AES-256-GCM
type Crypto struct {
	storeDir      string
	passwordCache *cache.PasswordCache
}

// New creates a new Crypto instance
func New(storeDir string) *Crypto {
	// Default cache timeout of 5 minutes
	passwordCache := cache.NewPasswordCache(storeDir, 5*time.Minute)
	return &Crypto{
		storeDir:      storeDir,
		passwordCache: passwordCache,
	}
}

// NewFromStore creates a Crypto handler for an existing store
func NewFromStore(storeDir string) (*Crypto, error) {
	// Check if store is initialized by looking for any encrypted files or config
	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("password store directory does not exist")
	}

	// Default cache timeout of 5 minutes
	passwordCache := cache.NewPasswordCache(storeDir, 5*time.Minute)
	return &Crypto{
		storeDir:      storeDir,
		passwordCache: passwordCache,
	}, nil
}

// Encrypt encrypts data using a master password with Argon2id + AES-256-GCM
func (c *Crypto) Encrypt(data []byte, masterPassword string) ([]byte, error) {
	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using Argon2id
	key := argon2.IDKey([]byte(masterPassword), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Combine salt, nonce, and ciphertext
	result := make([]byte, 0, saltSize+nonceSize+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt decrypts data using a master password
func (c *Crypto) Decrypt(encryptedData []byte, masterPassword string) ([]byte, error) {
	if len(encryptedData) < saltSize+nonceSize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	// Extract salt, nonce, and ciphertext
	salt := encryptedData[:saltSize]
	nonce := encryptedData[saltSize : saltSize+nonceSize]
	ciphertext := encryptedData[saltSize+nonceSize:]

	// Derive key using Argon2id with the same salt
	key := argon2.IDKey([]byte(masterPassword), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data (wrong password?): %w", err)
	}

	return plaintext, nil
}

// PromptMasterPassword securely prompts for the master password with caching
func (c *Crypto) PromptMasterPassword(prompt string) (string, error) {
	// Check if we have a cached password first
	if cachedPassword, found := c.passwordCache.Get(); found {
		// Validate the cached password
		if err := c.ValidateMasterPassword(cachedPassword); err == nil {
			return cachedPassword, nil
		}
		// If cached password is invalid, clear the cache
		c.passwordCache.Clear()
	}

	fmt.Print(prompt)

	// Read password without echoing to terminal
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after password input

	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	password := string(passwordBytes)

	// Validate the password before caching
	if err := c.ValidateMasterPassword(password); err != nil {
		return "", err
	}

	// Cache the valid password
	if err := c.passwordCache.Set(password); err != nil {
		// Log the error but don't fail - caching is not critical
		fmt.Printf("Warning: failed to cache password: %v\n", err)
	}

	return password, nil
}

// InitializeMasterPassword sets up the master password for a new store
func (c *Crypto) InitializeMasterPassword(masterPassword string) error {
	masterFile := filepath.Join(c.storeDir, masterPasswordFile)

	// Check if master password file already exists
	if _, err := os.Stat(masterFile); err == nil {
		return fmt.Errorf("store already initialized with a master password")
	}

	// Generate random salt for master password
	salt := make([]byte, masterSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt for master password: %w", err)
	}

	// Hash master password with Argon2id
	hash := argon2.IDKey([]byte(masterPassword), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Combine salt and hash
	masterData := make([]byte, 0, masterSaltSize+argon2KeyLen)
	masterData = append(masterData, salt...)
	masterData = append(masterData, hash...)

	// Write to master password file
	if err := os.WriteFile(masterFile, masterData, 0600); err != nil {
		return fmt.Errorf("failed to write master password file: %w", err)
	}

	return nil
}

// ValidateMasterPassword checks if the provided password matches the stored master password
func (c *Crypto) ValidateMasterPassword(masterPassword string) error {
	masterFile := filepath.Join(c.storeDir, masterPasswordFile)

	// Read master password file
	masterData, err := os.ReadFile(masterFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("store not initialized. Run 'pwd-mngr init' first")
		}
		return fmt.Errorf("failed to read master password file: %w", err)
	}

	if len(masterData) != masterSaltSize+argon2KeyLen {
		return fmt.Errorf("invalid master password file format")
	}

	// Extract salt and stored hash
	salt := masterData[:masterSaltSize]
	storedHash := masterData[masterSaltSize:]

	// Hash the provided password with the same salt
	providedHash := argon2.IDKey([]byte(masterPassword), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Compare hashes (constant-time comparison)
	if len(providedHash) != len(storedHash) {
		return fmt.Errorf("invalid master password")
	}

	// Constant-time comparison to prevent timing attacks
	result := byte(0)
	for i := 0; i < len(storedHash); i++ {
		result |= storedHash[i] ^ providedHash[i]
	}

	if result != 0 {
		return fmt.Errorf("invalid master password")
	}

	return nil
}

// IsStoreInitialized checks if the store has been initialized with a master password
func (c *Crypto) IsStoreInitialized() bool {
	masterFile := filepath.Join(c.storeDir, masterPasswordFile)
	_, err := os.Stat(masterFile)
	return err == nil
}

// ValidateStoreAccess checks if we can access the store with the given master password
func (c *Crypto) ValidateStoreAccess(masterPassword string) error {
	return c.ValidateMasterPassword(masterPassword)
}

// ClearPasswordCache clears the cached master password
func (c *Crypto) ClearPasswordCache() {
	c.passwordCache.Clear()
}

// SetCacheTimeout sets the cache timeout duration
func (c *Crypto) SetCacheTimeout(timeout time.Duration) {
	c.passwordCache.SetTimeout(timeout)
}

// GetCacheRemainingTime returns the remaining time before cache expiration
func (c *Crypto) GetCacheRemainingTime() time.Duration {
	return c.passwordCache.GetRemainingTime()
}

// IsCacheValid checks if the password cache is valid and not expired
func (c *Crypto) IsCacheValid() bool {
	if cachedPassword, found := c.passwordCache.Get(); found {
		return c.ValidateMasterPassword(cachedPassword) == nil
	}
	return false
}
