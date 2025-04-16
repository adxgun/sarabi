package misc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// Encryptor handles encryption and decryption of application variables/secrets
type Encryptor interface {
	GenerateKey() ([32]byte, error)
	Encrypt(data string) (string, error)
	Decrypt(data string) (string, error)
}

const (
	encryptionKeyFile = "sarabi.aes"
)

type encryptor struct {
	encryptionKey string
}

func NewEncryptor(key string) Encryptor {
	return &encryptor{
		encryptionKey: key,
	}
}

// GenerateKey generates a random [32]byte encryption key and stores it in /sarabi_path/sarabi.aes
// this function checks for an existing key and return that if available.
// the generated key should be kept safe outside of this server(aka storage) in-case of data loss, in order to be able to retrieve
// encrypted data(mainly application variables/secrets)
func (e *encryptor) GenerateKey() ([32]byte, error) {
	var key [32]byte
	if e.encryptionKey != "" {
		return sha256.Sum256([]byte(e.encryptionKey)), nil
	}

	// Try reading from key file if it exists
	if data, err := os.ReadFile(encryptionKeyFile); err == nil {
		decoded, err := hex.DecodeString(string(data))
		if err != nil {
			return key, fmt.Errorf("failed to decode key from file: %w", err)
		}
		if len(decoded) != 32 {
			return key, fmt.Errorf("invalid key length from file: got %d bytes, want 32", len(decoded))
		}
		copy(key[:], decoded)
		return key, nil
	}

	// Generate new key
	randomKey := make([]byte, 32)
	if _, err := rand.Read(randomKey); err != nil {
		return key, fmt.Errorf("failed to generate key: %w", err)
	}

	// Save new key to file
	keyHex := hex.EncodeToString(randomKey)
	if err := os.WriteFile(encryptionKeyFile, []byte(keyHex), 0600); err != nil {
		return key, fmt.Errorf("failed to write key to file: %w", err)
	}

	copy(key[:], randomKey)
	return key, nil
}

func (e *encryptor) Encrypt(data string) (string, error) {
	key, err := e.GenerateKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(data), nil)
	return hex.EncodeToString(ciphertext), nil
}

func (e *encryptor) Decrypt(data string) (string, error) {
	key, err := e.GenerateKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := hex.DecodeString(data)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
