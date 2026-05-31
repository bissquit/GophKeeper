// Package crypto implements client-side payload encryption for GophKeeper
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

// KeySize is the AES-256 key length in bytes
const KeySize = 32

// ErrCiphertextTooShort is returned when ciphertext is missing nonce or auth tag
var ErrCiphertextTooShort = errors.New("ciphertext too short")

// DeriveKey returns a 32-byte AES key derived from the master password
func DeriveKey(masterPassword, login string) []byte {
	saltSum := sha256.Sum256([]byte("gophkeeper:" + login))
	return argon2.IDKey([]byte(masterPassword), saltSum[:], 1, 64*1024, 4, KeySize)
}

// Encrypt seals plaintext with AES-256-GCM, prepending a fresh random nonce
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt opens AES-256-GCM ciphertext produced by Encrypt
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, ErrCiphertextTooShort
	}
	nonce, sealed := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, sealed, nil)
}
