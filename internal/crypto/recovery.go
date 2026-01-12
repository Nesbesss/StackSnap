package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)





const RecoveryKitVersion = 1


type RecoveryKit struct {
	Version   int  `json:"version"`
	Salt     string `json:"salt"`
	EncryptedKey string `json:"encrypted_key"`
	Hint     string `json:"hint,omitempty"`
}


const (
	argon2Time  = 3
	argon2Memory = 64 * 1024
	argon2Threads = 4
	argon2KeyLen = 32
)



func CreateRecoveryKit(backupKey []byte, passphrase string, hint string) (*RecoveryKit, error) {
	if len(passphrase) < 8 {
		return nil, fmt.Errorf("passphrase must be at least 8 characters")
	}


	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}


	derivedKey := argon2.IDKey(
		[]byte(passphrase),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)


	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}


	ciphertext := aead.Seal(nonce, nonce, backupKey, nil)

	return &RecoveryKit{
		Version:   RecoveryKitVersion,
		Salt:     hex.EncodeToString(salt),
		EncryptedKey: hex.EncodeToString(ciphertext),
		Hint:     hint,
	}, nil
}


func RecoverKey(kit *RecoveryKit, passphrase string) ([]byte, error) {
	if kit.Version != RecoveryKitVersion {
		return nil, fmt.Errorf("unsupported recovery kit version: %d", kit.Version)
	}


	salt, err := hex.DecodeString(kit.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %w", err)
	}

	ciphertext, err := hex.DecodeString(kit.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted key: %w", err)
	}


	derivedKey := argon2.IDKey(
		[]byte(passphrase),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)


	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:aead.NonceSize()]
	encrypted := ciphertext[aead.NonceSize():]

	backupKey, err := aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong passphrase?): %w", err)
	}

	return backupKey, nil
}


func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}


func KeyToHex(key []byte) string {
	return hex.EncodeToString(key)
}


func KeyFromHex(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	return key, nil
}



func DeriveKeyFromPassword(password string, salt []byte) []byte {
	if salt == nil {
		salt = make([]byte, 16)


		copy(salt, []byte("stacksnap-salt!"))
	}

	return argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)
}


func ValidateKey(key []byte) error {
	if key == nil {
		return fmt.Errorf("key is nil")
	}
	if len(key) != 32 {
		return fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	if bytes.Equal(key, make([]byte, 32)) {
		return fmt.Errorf("key cannot be all zeros")
	}
	return nil
}
