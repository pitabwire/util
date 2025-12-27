package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
)

// ComputeLookupToken generates a cryptographically secure lookup token from input data.
//
// The token is computed using HMAC-SHA256 with the provided key, making it suitable for:
// - Database indexing operations
// - Cache key generation
// - Deduplication identifiers
//
// Security properties:
//   - Deterministic: Same input always produces the same token
//   - Non-reversible: Cannot derive the original input from the token
//   - Constant-time comparison: Safe against timing attacks
//   - Rainbow table resistant: Requires secret HMAC key
//
// The function is tenant-scoped when tenant_id is included in the normalized input,
// ensuring multi-tenant data isolation.
//
// Parameters:
//   - hmacKey: Secret key for HMAC (must be kept secure, recommended 32+ bytes)
//   - normalized: Input data to be tokenized (should be pre-normalized for consistency)
//
// Returns:
//   - 32-byte HMAC-SHA256 token suitable for indexing and comparison
//
// Example:
//
//	key := []byte("32-byte-secret-key-for-hmac")
//	input := "user123@example.com"
//	token := ComputeLookupToken(key, input)
func ComputeLookupToken(hmacKey []byte, normalized string) []byte {
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write([]byte(normalized))
	return mac.Sum(nil)
}

// EncryptValue encrypts plaintext using AES-GCM with authenticated encryption.
//
// AES-GCM (Galois/Counter Mode) provides both confidentiality and authenticity,
// making it suitable for sensitive data encryption. The function generates a
// random nonce for each encryption to ensure semantic security.
//
// Security properties:
//   - Confidentiality: Plaintext is encrypted using AES-256
//   - Authenticity: GCM authentication tag detects tampering
//   - Semantic security: Random nonce ensures identical plaintexts
//     produce different ciphertexts
//   - No padding oracle: GCM doesn't require padding
//
// Parameters:
//   - aesKey: AES encryption key (must be 16, 24, or 32 bytes for AES-128/192/256)
//   - plaintext: Data to be encrypted
//
// Returns:
//   - Combined nonce + ciphertext (nonce size + ciphertext + auth tag)
//   - Error if key is invalid or encryption fails
//
// The returned payload format is: [nonce][ciphertext][authentication-tag]
// Use DecryptValue with the same key to decrypt.
//
// Example:
//
//	key := make([]byte, 32) // AES-256 key
//	rand.Read(key)
//	ciphertext, err := EncryptValue(key, []byte("sensitive data"))
func EncryptValue(aesKey []byte, plaintext []byte) ([]byte, error) {
	if len(aesKey) != 16 && len(aesKey) != 24 && len(aesKey) != 32 {
		return nil, errors.New("AES key must be 16, 24, or 32 bytes long")
	}

	if len(plaintext) == 0 {
		return nil, errors.New("plaintext cannot be empty")
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result, nonce)
	copy(result[len(nonce):], ciphertext)
	return result, nil
}

// DecryptValue decrypts data encrypted with EncryptValue using AES-GCM.
//
// This function verifies the authentication tag to ensure the ciphertext
// has not been tampered with before decryption. Any modification to the
// ciphertext or nonce will cause decryption to fail.
//
// Security properties:
//   - Authenticated decryption: Detects any ciphertext tampering
//   - Automatic nonce extraction: Nonce is read from the payload prefix
//   - Constant-time operations: Safe against timing attacks in verification
//
// Parameters:
//   - aesKey: AES decryption key (must be identical to encryption key)
//   - payload: Combined nonce + ciphertext from EncryptValue
//
// Returns:
//   - Decrypted plaintext data
//   - Error if decryption fails, authentication fails, or inputs are invalid
//
// Common failure scenarios:
//   - Incorrect key: "cipher: message authentication failed"
//   - Corrupted payload: "cipher: message authentication failed"
//   - Invalid payload length: Various cipher errors
//   - Wrong key size: "crypto/aes: invalid key size"
//
// Example:
//
//	plaintext, err := DecryptValue(key, ciphertext)
//	if err != nil {
//	    // Handle decryption failure
//	}
func DecryptValue(aesKey []byte, payload []byte) ([]byte, error) {
	if len(aesKey) != 16 && len(aesKey) != 24 && len(aesKey) != 32 {
		return nil, errors.New("AES key must be 16, 24, or 32 bytes long")
	}

	if len(payload) == 0 {
		return nil, errors.New("payload cannot be empty")
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(payload) < nonceSize {
		return nil, errors.New("payload too short to contain nonce")
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]

	if len(ciphertext) == 0 {
		return nil, errors.New("payload contains no ciphertext")
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}
