package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"chowkidaar/internal/cache"

	"github.com/tyler-smith/go-bip39"
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

	// Keyfile for two-factor encryption
	keyFileName = ".keyfile"
	keyFileSize = 32 // 256 bits
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

	// Get combined key (password + keyfile)
	combinedKey, err := c.getCombinedKey(masterPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to get combined key: %w", err)
	}

	// Derive key using Argon2id
	key := argon2.IDKey(combinedKey, salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

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

	// Get combined key (password + keyfile)
	combinedKey, err := c.getCombinedKey(masterPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to get combined key: %w", err)
	}

	// Derive key using Argon2id with the same salt
	key := argon2.IDKey(combinedKey, salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

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
		// Return cached password (it was validated when first cached)
		return cachedPassword, nil
	}

	// Display full-screen banner
	leftPad, inputRow, boxWidth := c.displayPasswordBanner(prompt)

	// Read password with asterisk feedback
	password, err := c.readPasswordWithFeedback(leftPad, inputRow, boxWidth)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	// Clear the banner
	c.clearPasswordBanner()

	// Note: We don't cache the password here.
	// It will be cached only after successful decryption/encryption.
	// The caller should call CachePassword() after successful operation.

	return password, nil
}

// readPasswordWithFeedback reads password from stdin with asterisk visual feedback
// Asterisks are center-aligned within the box
func (c *Crypto) readPasswordWithFeedback(leftPad, row, boxWidth int) (string, error) {
	// Set terminal to raw mode to read individual characters
	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(syscall.Stdin), oldState)

	var password []byte
	var buf [1]byte

	// Helper function to redraw centered asterisks
	redrawPassword := func() {
		// Clear the input line
		fmt.Printf("\033[%d;%dH", row, leftPad+2)
		fmt.Print(strings.Repeat(" ", boxWidth-2))

		// Calculate centered position for asterisks
		asterisks := strings.Repeat("*", len(password))
		padding := (boxWidth - 2 - len(asterisks)) / 2
		if padding < 0 {
			padding = 0
		}

		// Position cursor and print centered asterisks
		fmt.Printf("\033[%d;%dH", row, leftPad+2+padding)
		fmt.Print(asterisks)
	}

	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}

		char := buf[0]

		// Handle special keys
		switch char {
		case 3: // Ctrl+C
			fmt.Println()
			return "", fmt.Errorf("interrupted")
		case 13, 10: // Enter (CR or LF)
			return string(password), nil
		case 127, 8: // Backspace or Delete
			if len(password) > 0 {
				password = password[:len(password)-1]
				redrawPassword()
			}
		default:
			// Add character to password
			if char >= 32 && char <= 126 { // Printable ASCII
				password = append(password, char)
				redrawPassword()
			}
		}
	}
}

// displayPasswordBanner shows a full-screen banner for password entry
// Returns (leftPadding, inputRow, boxWidth) for cursor positioning
func (c *Crypto) displayPasswordBanner(prompt string) (int, int, int) {
	// Clear screen
	fmt.Print("\033[2J\033[H")

	// Get terminal size
	width := 80  // default width
	height := 24 // default height
	if w, h, err := term.GetSize(int(syscall.Stdin)); err == nil && w > 0 {
		width = w
		height = h
	}

	// Calculate vertical centering
	topPadding := (height - 13) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Add top padding
	for i := 0; i < topPadding; i++ {
		fmt.Println()
	}

	// Calculate box width (max 60 chars or terminal width - 20)
	boxWidth := 60
	if width-20 < boxWidth {
		boxWidth = width - 20
	}
	if boxWidth < 40 {
		boxWidth = 40
	}

	// Calculate left padding for horizontal centering
	leftPadding := (width - boxWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}
	indent := strings.Repeat(" ", leftPadding)

	// Clean, professional banner with input line inside
	fmt.Println(indent + "┌" + strings.Repeat("─", boxWidth-2) + "┐")
	fmt.Println(indent + "│" + strings.Repeat(" ", boxWidth-2) + "│")
	fmt.Println(indent + "│" + centerText("CHOWKIDAAR", boxWidth-2) + "│")
	fmt.Println(indent + "│" + centerText("Password Manager", boxWidth-2) + "│")
	fmt.Println(indent + "│" + strings.Repeat(" ", boxWidth-2) + "│")
	fmt.Println(indent + "├" + strings.Repeat("─", boxWidth-2) + "┤")
	fmt.Println(indent + "│" + strings.Repeat(" ", boxWidth-2) + "│")
	fmt.Println(indent + "│" + centerText(prompt, boxWidth-2) + "│")
	fmt.Println(indent + "│" + strings.Repeat(" ", boxWidth-2) + "│")

	// Input line inside the box
	inputRow := topPadding + 10 // Row where input will appear
	fmt.Println(indent + "│" + strings.Repeat(" ", boxWidth-2) + "│")
	fmt.Println(indent + "│" + strings.Repeat(" ", boxWidth-2) + "│")
	fmt.Println(indent + "└" + strings.Repeat("─", boxWidth-2) + "┘")

	return leftPadding, inputRow, boxWidth
}

// clearPasswordBanner clears the password banner from screen
func (c *Crypto) clearPasswordBanner() {
	// Clear screen and return to normal
	fmt.Print("\033[2J\033[H")
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	// Count actual character width (accounting for emojis and Unicode)
	textLen := len([]rune(text))
	if textLen >= width {
		return text[:width]
	}
	padding := (width - textLen) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-textLen-padding)
}

// CachePassword caches a validated master password
func (c *Crypto) CachePassword(password string) error {
	return c.passwordCache.Set(password)
}

// GenerateMnemonic creates a new 12-word BIP-39 mnemonic phrase
func (c *Crypto) GenerateMnemonic() (string, error) {
	// Generate 128 bits of entropy (12 words)
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	// Convert to mnemonic
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	return mnemonic, nil
}

// CreateKeyFileFromMnemonic generates and saves a keyfile from a BIP-39 mnemonic
func (c *Crypto) CreateKeyFileFromMnemonic(mnemonic string) error {
	// Validate mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		return fmt.Errorf("invalid mnemonic phrase")
	}

	// Convert mnemonic to seed (we use empty passphrase)
	seed := bip39.NewSeed(mnemonic, "")

	// Use first 32 bytes as keyfile
	if len(seed) < keyFileSize {
		return fmt.Errorf("seed too short")
	}
	keyFileData := seed[:keyFileSize]

	// Write keyfile
	keyFilePath := filepath.Join(c.storeDir, keyFileName)
	if err := os.WriteFile(keyFilePath, keyFileData, 0600); err != nil {
		return fmt.Errorf("failed to write keyfile: %w", err)
	}

	return nil
}

// HasKeyFile checks if the keyfile exists
func (c *Crypto) HasKeyFile() bool {
	keyFilePath := filepath.Join(c.storeDir, keyFileName)
	_, err := os.Stat(keyFilePath)
	return err == nil
}

// HasEncryptedPasswords checks if any .enc files exist (indicating initialized store)
func (c *Crypto) HasEncryptedPasswords() (bool, error) {
	found := false
	err := filepath.Walk(c.storeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".enc") {
			fmt.Println(info.Name())
			found = true
			return filepath.SkipDir // Stop walking once we find one
		}
		return nil
	})

	if err != nil {
		return false, fmt.Errorf("failed to check for encrypted files: %w", err)
	}

	return found, nil
}

// getCombinedKey combines the master password with the keyfile
func (c *Crypto) getCombinedKey(masterPassword string) ([]byte, error) {
	// Read keyfile
	keyFilePath := filepath.Join(c.storeDir, keyFileName)
	keyFileData, err := os.ReadFile(keyFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("keyfile not found. Run 'chowkidaar init' first")
		}
		return nil, fmt.Errorf("failed to read keyfile: %w", err)
	}

	if len(keyFileData) != keyFileSize {
		return nil, fmt.Errorf("invalid keyfile size")
	}

	// Combine password and keyfile
	combined := make([]byte, 0, len(masterPassword)+keyFileSize)
	combined = append(combined, []byte(masterPassword)...)
	combined = append(combined, keyFileData...)

	return combined, nil
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
	_, found := c.passwordCache.Get()
	return found
}
