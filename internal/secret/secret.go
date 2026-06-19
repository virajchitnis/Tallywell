// Package secret implements Tallywell's at-rest encryption: AES-256-GCM blob
// encryption under a data-encryption key (DEK), and a multi-wrap key envelope
// that lets the same DEK be unlocked by any of several methods (passphrase, OS
// keychain, and — in future — a passkey). Adding an unlock method is just
// adding another wrap of the same DEK, so protection tiers and "add another
// unlock method" are the same operation.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// KeyLen is the length in bytes of a DEK and of any wrapping key (AES-256).
const KeyLen = 32

// ErrDecrypt is returned when decryption/authentication fails (wrong key or
// tampered ciphertext). It is deliberately generic to avoid leaking detail.
var ErrDecrypt = errors.New("secret: decryption failed")

// randBytes returns n cryptographically-random bytes.
func randBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("secret: read random: %w", err)
	}
	return b, nil
}

// GenerateKey returns a fresh random 32-byte key suitable for use as a DEK or
// a wrapping key.
func GenerateKey() ([]byte, error) {
	return randBytes(KeyLen)
}

// newGCM builds an AES-256-GCM AEAD for the key, validating its length. The
// aes/cipher constructors cannot actually fail for a valid 32-byte key, so the
// only reachable error here is the length check.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeyLen {
		return nil, fmt.Errorf("secret: key must be %d bytes", KeyLen)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// seal encrypts plaintext with AES-256-GCM under key, returning nonce||ciphertext.
func seal(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce, err := randBytes(gcm.NonceSize())
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// open decrypts a nonce||ciphertext blob produced by seal.
func open(key, blob []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, ErrDecrypt
	}
	nonce, ct := blob[:ns], blob[ns:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, ErrDecrypt
	}
	return pt, nil
}

// EncryptBlob encrypts arbitrary data (e.g. a database snapshot) under the DEK.
func EncryptBlob(dek, plaintext []byte) ([]byte, error) {
	return seal(dek, plaintext)
}

// DecryptBlob reverses EncryptBlob; it returns ErrDecrypt on a wrong key or
// tampered data.
func DecryptBlob(dek, blob []byte) ([]byte, error) {
	return open(dek, blob)
}
