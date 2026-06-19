package secret

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptBlobRoundTrip(t *testing.T) {
	dek, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("some database snapshot bytes")
	blob, err := EncryptBlob(dek, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(blob, plaintext) {
		t.Fatal("ciphertext should not contain plaintext")
	}
	got, err := DecryptBlob(dek, blob)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round trip mismatch: %q", got)
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	dek, _ := GenerateKey()
	other, _ := GenerateKey()
	blob, _ := EncryptBlob(dek, []byte("secret"))
	if _, err := DecryptBlob(other, blob); err != ErrDecrypt {
		t.Fatalf("expected ErrDecrypt, got %v", err)
	}
}

func TestDecryptTamperedFails(t *testing.T) {
	dek, _ := GenerateKey()
	blob, _ := EncryptBlob(dek, []byte("secret payload"))
	blob[len(blob)-1] ^= 0xFF // flip a ciphertext bit
	if _, err := DecryptBlob(dek, blob); err != ErrDecrypt {
		t.Fatalf("expected ErrDecrypt on tamper, got %v", err)
	}
}

func TestDecryptTooShortFails(t *testing.T) {
	dek, _ := GenerateKey()
	if _, err := DecryptBlob(dek, []byte("x")); err != ErrDecrypt {
		t.Fatalf("expected ErrDecrypt, got %v", err)
	}
}

func TestBadKeyLength(t *testing.T) {
	if _, err := EncryptBlob([]byte("short"), []byte("data")); err == nil {
		t.Fatal("expected error for short key")
	}
}
