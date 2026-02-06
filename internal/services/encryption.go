package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/DevAnuragT/context_keeper/internal/config"
)

// EncryptionService handles encryption and decryption of sensitive data
type EncryptionService interface {
	Encrypt(ctx context.Context, plaintext string) (string, error)
	Decrypt(ctx context.Context, ciphertext string) (string, error)
	EncryptMap(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)
	DecryptMap(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)
}

// EncryptionServiceImpl implements the EncryptionService interface
type EncryptionServiceImpl struct {
	key []byte
}

// NewEncryptionService creates a new encryption service
func NewEncryptionService(cfg *config.Config) EncryptionService {
	// Use JWT secret as encryption key base
	hash := sha256.Sum256([]byte(cfg.JWTSecret))
	return &EncryptionServiceImpl{
		key: hash[:],
	}
}

// Encrypt encrypts a plaintext string using AES-GCM
func (e *EncryptionServiceImpl) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Create AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a ciphertext string using AES-GCM
func (e *EncryptionServiceImpl) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum length
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext_bytes := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext_bytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptMap encrypts sensitive values in a map
func (e *EncryptionServiceImpl) EncryptMap(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	if data == nil {
		return nil, nil
	}

	result := make(map[string]interface{})
	
	for key, value := range data {
		if e.isSensitiveField(key) {
			// Encrypt sensitive fields
			if strValue, ok := value.(string); ok && strValue != "" {
				encrypted, err := e.Encrypt(ctx, strValue)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt field %s: %w", key, err)
				}
				result[key] = encrypted
			} else {
				result[key] = value
			}
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			// Recursively encrypt nested maps
			encrypted, err := e.EncryptMap(ctx, nestedMap)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt nested map in field %s: %w", key, err)
			}
			result[key] = encrypted
		} else {
			// Copy non-sensitive fields as-is
			result[key] = value
		}
	}

	return result, nil
}

// DecryptMap decrypts sensitive values in a map
func (e *EncryptionServiceImpl) DecryptMap(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	if data == nil {
		return nil, nil
	}

	result := make(map[string]interface{})
	
	for key, value := range data {
		if e.isSensitiveField(key) {
			// Decrypt sensitive fields
			if strValue, ok := value.(string); ok && strValue != "" {
				decrypted, err := e.Decrypt(ctx, strValue)
				if err != nil {
					return nil, fmt.Errorf("failed to decrypt field %s: %w", key, err)
				}
				result[key] = decrypted
			} else {
				result[key] = value
			}
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			// Recursively decrypt nested maps
			decrypted, err := e.DecryptMap(ctx, nestedMap)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt nested map in field %s: %w", key, err)
			}
			result[key] = decrypted
		} else {
			// Copy non-sensitive fields as-is
			result[key] = value
		}
	}

	return result, nil
}

// isSensitiveField determines if a field should be encrypted
func (e *EncryptionServiceImpl) isSensitiveField(fieldName string) bool {
	sensitiveFields := map[string]bool{
		"access_token":    true,
		"refresh_token":   true,
		"client_secret":   true,
		"api_key":         true,
		"private_key":     true,
		"password":        true,
		"secret":          true,
		"token":           true,
		"webhook_secret":  true,
		"bot_token":       true,
		"oauth_token":     true,
		"credentials":     true,
	}

	return sensitiveFields[fieldName]
}

// MockEncryptionService is a mock implementation for testing
type MockEncryptionService struct {
	EncryptFunc    func(ctx context.Context, plaintext string) (string, error)
	DecryptFunc    func(ctx context.Context, ciphertext string) (string, error)
	EncryptMapFunc func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)
	DecryptMapFunc func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)
}

// NewMockEncryptionService creates a new mock encryption service
func NewMockEncryptionService() *MockEncryptionService {
	return &MockEncryptionService{
		EncryptFunc: func(ctx context.Context, plaintext string) (string, error) {
			if plaintext == "" {
				return "", nil
			}
			return "encrypted_" + plaintext, nil
		},
		DecryptFunc: func(ctx context.Context, ciphertext string) (string, error) {
			if ciphertext == "" {
				return "", nil
			}
			if len(ciphertext) > 10 && ciphertext[:10] == "encrypted_" {
				return ciphertext[10:], nil
			}
			return ciphertext, nil
		},
		EncryptMapFunc: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
			if data == nil {
				return nil, nil
			}
			result := make(map[string]interface{})
			for k, v := range data {
				if strValue, ok := v.(string); ok && (k == "access_token" || k == "secret" || k == "password") {
					result[k] = "encrypted_" + strValue
				} else {
					result[k] = v
				}
			}
			return result, nil
		},
		DecryptMapFunc: func(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
			if data == nil {
				return nil, nil
			}
			result := make(map[string]interface{})
			for k, v := range data {
				if strValue, ok := v.(string); ok && len(strValue) > 10 && strValue[:10] == "encrypted_" {
					result[k] = strValue[10:]
				} else {
					result[k] = v
				}
			}
			return result, nil
		},
	}
}

func (m *MockEncryptionService) Encrypt(ctx context.Context, plaintext string) (string, error) {
	return m.EncryptFunc(ctx, plaintext)
}

func (m *MockEncryptionService) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	return m.DecryptFunc(ctx, ciphertext)
}

func (m *MockEncryptionService) EncryptMap(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	return m.EncryptMapFunc(ctx, data)
}

func (m *MockEncryptionService) DecryptMap(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	return m.DecryptMapFunc(ctx, data)
}