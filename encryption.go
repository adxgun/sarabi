package sarabi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// Encryptor handles encryption and decryption of application variables/secrets
type Encryptor interface {
	GenerateKey() ([]byte, error)
	Encrypt(data string) (string, error)
	Decrypt(data string) (string, error)
}

const (
	encryptionKeyFile = "sarabi.aes"
)

type encryptor struct{}

func NewEncryptor() Encryptor {
	return &encryptor{}
}

// GenerateKey generates a random [32]byte encryption key and stores it in /sarabi_path/sarabi.aes
// this function checks for an existing key and return that if available.
// the generated key should be kept safe outside of this server(aka storage) in-case of data loss, in order to be able to retrieve
// encrypted data(mainly application variables/secrets)
func (e *encryptor) GenerateKey() ([]byte, error) {
	if _, err := os.Stat(encryptionKeyFile); err == nil {
		f, err := os.Open(encryptionKeyFile)
		if err != nil {
			return nil, err
		}

		content, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}
		return hex.DecodeString(string(content))
	}

	handle, err := os.Create(encryptionKeyFile)
	if err != nil {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %v", err)
	}

	keyHex := hex.EncodeToString(key)
	_, err = io.WriteString(handle, keyHex)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (e *encryptor) Encrypt(data string) (string, error) {
	key, err := e.GenerateKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
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

	block, err := aes.NewCipher(key)
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
