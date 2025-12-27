package util_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"strings"
	"testing"

	"github.com/pitabwire/util"
)

func TestComputeLookupToken(t *testing.T) {
	tests := []struct {
		name       string
		hmacKey    []byte
		normalized string
		wantLen    int
	}{
		{
			name:       "valid input produces 32-byte token",
			hmacKey:    []byte("32-byte-secret-key-for-hmac-testing"),
			normalized: "user123@example.com",
			wantLen:    32,
		},
		{
			name:       "empty string produces valid token",
			hmacKey:    []byte("test-key-16-bytes-"),
			normalized: "",
			wantLen:    32,
		},
		{
			name:       "long input produces valid token",
			hmacKey:    []byte("test-key-16-bytes-"),
			normalized: strings.Repeat("a", 1000),
			wantLen:    32,
		},
		{
			name:       "unicode input produces valid token",
			hmacKey:    []byte("test-key-16-bytes-"),
			normalized: "用户123@例子.com",
			wantLen:    32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.ComputeLookupToken(tt.hmacKey, tt.normalized)
			if len(got) != tt.wantLen {
				t.Errorf("util.ComputeLookupToken() length = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestComputeLookupTokenDeterministic(t *testing.T) {
	key := []byte("test-key-16-bytes-")
	input := "test@example.com"

	token1 := util.ComputeLookupToken(key, input)
	token2 := util.ComputeLookupToken(key, input)

	if !bytes.Equal(token1, token2) {
		t.Error("util.ComputeLookupToken() should be deterministic")
	}
}

func TestComputeLookupTokenKeyIsolation(t *testing.T) {
	input := "test@example.com"
	key1 := []byte("test-key-16-bytes-")
	key2 := []byte("different-key-16-b")

	token1 := util.ComputeLookupToken(key1, input)
	token2 := util.ComputeLookupToken(key2, input)

	if bytes.Equal(token1, token2) {
		t.Error("util.ComputeLookupToken() should produce different tokens with different keys")
	}
}

func TestComputeLookupTokenInputIsolation(t *testing.T) {
	key := []byte("test-key-16-bytes-")
	input1 := "test1@example.com"
	input2 := "test2@example.com"

	token1 := util.ComputeLookupToken(key, input1)
	token2 := util.ComputeLookupToken(key, input2)

	if bytes.Equal(token1, token2) {
		t.Error("util.ComputeLookupToken() should produce different tokens with different inputs")
	}
}

func validateEncryptResult(t *testing.T, got []byte, plaintext []byte, err error, wantErr bool, errMsg string) {
	if (err != nil) != wantErr {
		t.Errorf("util.EncryptValue() error = %v, wantErr %v", err, wantErr)
		return
	}

	if wantErr && errMsg != "" && !strings.Contains(err.Error(), errMsg) {
		t.Errorf("util.EncryptValue() error = %v, expected to contain %v", err.Error(), errMsg)
		return
	}

	if !wantErr {
		if len(got) == 0 {
			t.Error("util.EncryptValue() returned empty ciphertext")
		}

		// Ciphertext should be longer than plaintext due to nonce and auth tag
		if len(got) <= len(plaintext) {
			t.Errorf(
				"util.EncryptValue() ciphertext length %d should be > plaintext length %d",
				len(got),
				len(plaintext),
			)
		}
	}
}

func TestEncryptValue(t *testing.T) {
	tests := []struct {
		name      string
		aesKey    []byte
		plaintext []byte
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid AES-128 key",
			aesKey:    make([]byte, 16),
			plaintext: []byte("test data"),
			wantErr:   false,
		},
		{
			name:      "valid AES-192 key",
			aesKey:    make([]byte, 24),
			plaintext: []byte("test data"),
			wantErr:   false,
		},
		{
			name:      "valid AES-256 key",
			aesKey:    make([]byte, 32),
			plaintext: []byte("test data"),
			wantErr:   false,
		},
		{
			name:      "invalid key size",
			aesKey:    make([]byte, 10),
			plaintext: []byte("test data"),
			wantErr:   true,
			errMsg:    "AES key must be 16, 24, or 32 bytes long",
		},
		{
			name:      "empty plaintext",
			aesKey:    make([]byte, 32),
			plaintext: []byte{},
			wantErr:   true,
			errMsg:    "plaintext cannot be empty",
		},
		{
			name:      "large plaintext",
			aesKey:    make([]byte, 32),
			plaintext: bytes.Repeat([]byte("a"), 10000),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr && tt.name != "large plaintext" {
				rand.Read(tt.aesKey)
			}

			got, err := util.EncryptValue(tt.aesKey, tt.plaintext)
			validateEncryptResult(t, got, tt.plaintext, err, tt.wantErr, tt.errMsg)
		})
	}
}

func TestEncryptValueSemanticSecurity(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	plaintext := []byte("same plaintext")

	ciphertext1, err := util.EncryptValue(key, plaintext)
	if err != nil {
		t.Fatalf("util.EncryptValue() failed: %v", err)
	}

	ciphertext2, err := util.EncryptValue(key, plaintext)
	if err != nil {
		t.Fatalf("util.EncryptValue() failed: %v", err)
	}

	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("util.EncryptValue() should produce different ciphertexts for same plaintext")
	}
}

func validateDecryptResult(t *testing.T, got []byte, expectedPlain []byte, err error, wantErr bool, errMsg string) {
	if (err != nil) != wantErr {
		t.Errorf("DecryptValue() error = %v, wantErr %v", err, wantErr)
		return
	}

	if wantErr && errMsg != "" && !strings.Contains(err.Error(), errMsg) {
		t.Errorf("DecryptValue() error = %v, expected to contain %v", err.Error(), errMsg)
		return
	}

	if !wantErr {
		if !bytes.Equal(got, expectedPlain) {
			t.Errorf("DecryptValue() = %v, want %v", got, expectedPlain)
		}
	}
}

func TestDecryptValue(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() ([]byte, []byte, []byte, error) // returns key, payload, expected plaintext, error
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid decryption",
			setup: func() ([]byte, []byte, []byte, error) {
				key := make([]byte, 32)
				rand.Read(key)
				plaintext := []byte("test data for decryption")
				ciphertext, err := util.EncryptValue(key, plaintext)
				return key, ciphertext, plaintext, err
			},
			wantErr: false,
		},
		{
			name: "wrong key",
			setup: func() ([]byte, []byte, []byte, error) {
				key := make([]byte, 32)
				rand.Read(key)
				plaintext := []byte("test data for decryption")
				ciphertext, err := util.EncryptValue(key, plaintext)
				if err != nil {
					return nil, nil, nil, err
				}
				wrongKey := make([]byte, 32)
				rand.Read(wrongKey)
				return wrongKey, ciphertext, plaintext, nil
			},
			wantErr: true,
			errMsg:  "decryption failed",
		},
		{
			name: "invalid key size",
			setup: func() ([]byte, []byte, []byte, error) {
				key := make([]byte, 32)
				rand.Read(key)
				plaintext := []byte("test data for decryption")
				ciphertext, err := util.EncryptValue(key, plaintext)
				if err != nil {
					return nil, nil, nil, err
				}
				invalidKey := make([]byte, 10)
				return invalidKey, ciphertext, plaintext, nil
			},
			wantErr: true,
			errMsg:  "AES key must be 16, 24, or 32 bytes long",
		},
		{
			name: "empty payload",
			setup: func() ([]byte, []byte, []byte, error) {
				key := make([]byte, 32)
				rand.Read(key)
				return key, []byte{}, []byte{}, nil
			},
			wantErr: true,
			errMsg:  "payload cannot be empty",
		},
		{
			name: "payload too short",
			setup: func() ([]byte, []byte, []byte, error) {
				key := make([]byte, 32)
				rand.Read(key)
				return key, []byte{1, 2, 3}, []byte{}, nil
			},
			wantErr: true,
			errMsg:  "payload too short to contain nonce",
		},
		{
			name: "corrupted payload",
			setup: func() ([]byte, []byte, []byte, error) {
				key := make([]byte, 32)
				rand.Read(key)
				plaintext := []byte("test data for decryption")
				ciphertext, err := util.EncryptValue(key, plaintext)
				if err != nil {
					return nil, nil, nil, err
				}
				// Corrupt the payload by removing some bytes
				corrupted := make([]byte, len(ciphertext)-5)
				copy(corrupted, ciphertext[:10])
				copy(corrupted[10:], ciphertext[15:])
				return key, corrupted, plaintext, nil
			},
			wantErr: true,
			errMsg:  "decryption failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, payload, expectedPlain, setupErr := tt.setup()
			if setupErr != nil {
				t.Fatalf("Setup failed: %v", setupErr)
			}

			got, err := util.DecryptValue(key, payload)
			validateDecryptResult(t, got, expectedPlain, err, tt.wantErr, tt.errMsg)
		})
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	testCases := []struct {
		name      string
		keySize   int
		plaintext []byte
	}{
		{
			name:      "AES-128 with short text",
			keySize:   16,
			plaintext: []byte("hello world"),
		},
		{
			name:      "AES-192 with medium text",
			keySize:   24,
			plaintext: []byte("this is a medium length plaintext for testing"),
		},
		{
			name:      "AES-256 with long text",
			keySize:   32,
			plaintext: bytes.Repeat([]byte("long plaintext data "), 100),
		},
		{
			name:      "AES-256 with binary data",
			keySize:   32,
			plaintext: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		},
		{
			name:      "AES-256 with unicode",
			keySize:   32,
			plaintext: []byte("测试数据 🚀 Тест"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := make([]byte, tc.keySize)
			rand.Read(key)

			ciphertext, err := util.EncryptValue(key, tc.plaintext)
			if err != nil {
				t.Fatalf("util.EncryptValue() failed: %v", err)
			}

			decrypted, err := util.DecryptValue(key, ciphertext)
			if err != nil {
				t.Fatalf("DecryptValue() failed: %v", err)
			}

			if !bytes.Equal(decrypted, tc.plaintext) {
				t.Error("round trip failed: decrypted data doesn't match original")
			}
		})
	}
}

func TestMultipleEncryptionsUnique(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	plaintext := []byte("test data")

	ciphertexts := make([][]byte, 10)
	for i := range 10 {
		ct, err := util.EncryptValue(key, plaintext)
		if err != nil {
			t.Fatalf("util.EncryptValue() failed: %v", err)
		}
		ciphertexts[i] = ct
	}

	// Check all ciphertexts are unique
	for i := range ciphertexts {
		for j := i + 1; j < len(ciphertexts); j++ {
			if bytes.Equal(ciphertexts[i], ciphertexts[j]) {
				t.Errorf("ciphertexts %d and %d are identical", i, j)
			}
		}
	}

	// But all decrypt to the same plaintext
	for i, ct := range ciphertexts {
		decrypted, err := util.DecryptValue(key, ct)
		if err != nil {
			t.Fatalf("DecryptValue() failed for ciphertext %d: %v", i, err)
		}
		if !bytes.Equal(decrypted, plaintext) {
			t.Errorf("decrypted ciphertext %d doesn't match original", i)
		}
	}
}

// Benchmark tests.
func BenchmarkComputeLookupToken(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	input := "user123@example.com"

	b.ResetTimer()
	for range b.N {
		util.ComputeLookupToken(key, input)
	}
}

func BenchmarkEncryptValueAES128(b *testing.B) {
	key := make([]byte, 16)
	_, _ = rand.Read(key)
	plaintext := make([]byte, 1024)
	_, _ = rand.Read(plaintext)

	b.ResetTimer()
	for range b.N {
		_, _ = util.EncryptValue(key, plaintext)
	}
}

func BenchmarkEncryptValueAES256(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	plaintext := make([]byte, 1024)
	_, _ = rand.Read(plaintext)

	b.ResetTimer()
	for range b.N {
		_, _ = util.EncryptValue(key, plaintext)
	}
}

func BenchmarkDecryptValueAES128(b *testing.B) {
	key := make([]byte, 16)
	_, _ = rand.Read(key)
	plaintext := make([]byte, 1024)
	_, _ = rand.Read(plaintext)
	ciphertext, _ := util.EncryptValue(key, plaintext)

	b.ResetTimer()
	for range b.N {
		_, _ = util.DecryptValue(key, ciphertext)
	}
}

func BenchmarkDecryptValueAES256(b *testing.B) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	plaintext := make([]byte, 1024)
	_, _ = rand.Read(plaintext)
	ciphertext, _ := util.EncryptValue(key, plaintext)

	b.ResetTimer()
	for range b.N {
		_, _ = util.DecryptValue(key, ciphertext)
	}
}

// Example tests.
func ExampleComputeLookupToken() {
	key := []byte("32-byte-secret-key-for-hmac")
	input := "user123@example.com"
	token := util.ComputeLookupToken(key, input)

	// Token is a 32-byte array suitable for indexing
	fmt.Printf("%x", token)
	// Output: e0797b7f749ebb70773d3190feef382cc41d3e907485be7f5e3ee766c98463fc
}

func ExampleEncryptValue_roundtrip() {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	original := []byte("secret message")

	// Encrypt
	ciphertext, err := util.EncryptValue(key, original)
	if err != nil {
		panic(err)
	}

	// Decrypt
	decrypted, err := util.DecryptValue(key, ciphertext)
	if err != nil {
		panic(err)
	}

	// Verify round-trip
	if string(decrypted) != string(original) {
		panic("decryption failed")
	}

	fmt.Println(string(decrypted))
	// Output: secret message
}
