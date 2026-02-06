package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// PasswordService handles password hashing and verification
type PasswordService interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, hash string) (bool, error)
	GenerateSecureToken() (string, error)
}

// PasswordServiceImpl implements the PasswordService interface
type PasswordServiceImpl struct {
	// Argon2 parameters
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
	saltLen uint32
}

// NewPasswordService creates a new password service with secure defaults
func NewPasswordService() PasswordService {
	return &PasswordServiceImpl{
		time:    1,      // 1 iteration
		memory:  64 * 1024, // 64 MB
		threads: 4,      // 4 threads
		keyLen:  32,     // 32 bytes key length
		saltLen: 16,     // 16 bytes salt length
	}
}

// HashPassword hashes a password using Argon2id
func (p *PasswordServiceImpl) HashPassword(password string) (string, error) {
	// Generate a random salt
	salt := make([]byte, p.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, p.keyLen)

	// Encode the hash and salt
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	encodedHash := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		p.memory, p.time, p.threads, b64Salt, b64Hash)

	return encodedHash, nil
}

// VerifyPassword verifies a password against its hash
func (p *PasswordServiceImpl) VerifyPassword(password, encodedHash string) (bool, error) {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return false, fmt.Errorf("unsupported hash algorithm: %s", parts[1])
	}

	if parts[2] != "v=19" {
		return false, fmt.Errorf("unsupported argon2 version: %s", parts[2])
	}

	// Parse parameters
	var memory, time uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Hash the provided password with the same parameters
	keyLen := uint32(len(hash))
	comparisonHash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	// Compare hashes using constant-time comparison
	return subtle.ConstantTimeCompare(hash, comparisonHash) == 1, nil
}

// GenerateSecureToken generates a cryptographically secure random token
func (p *PasswordServiceImpl) GenerateSecureToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// MockPasswordService is a mock implementation for testing
type MockPasswordService struct {
	HashPasswordFunc     func(password string) (string, error)
	VerifyPasswordFunc   func(password, hash string) (bool, error)
	GenerateTokenFunc    func() (string, error)
}

// NewMockPasswordService creates a new mock password service
func NewMockPasswordService() *MockPasswordService {
	return &MockPasswordService{
		HashPasswordFunc: func(password string) (string, error) {
			return "mock_hash_" + password, nil
		},
		VerifyPasswordFunc: func(password, hash string) (bool, error) {
			return hash == "mock_hash_"+password, nil
		},
		GenerateTokenFunc: func() (string, error) {
			return "mock_token_123", nil
		},
	}
}

func (m *MockPasswordService) HashPassword(password string) (string, error) {
	return m.HashPasswordFunc(password)
}

func (m *MockPasswordService) VerifyPassword(password, hash string) (bool, error) {
	return m.VerifyPasswordFunc(password, hash)
}

func (m *MockPasswordService) GenerateSecureToken() (string, error) {
	return m.GenerateTokenFunc()
}