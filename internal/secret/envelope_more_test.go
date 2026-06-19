package secret

import "testing"

func TestMethodsList(t *testing.T) {
	e, dek, _ := NewEnvelopeWithPassphrase("pp")
	wk, _ := GenerateKey()
	_ = e.AddRawKey(dek, wk, MethodKeychain, "acct")

	methods := e.Methods()
	if len(methods) != 2 {
		t.Fatalf("expected 2 methods, got %v", methods)
	}
	seen := map[Method]bool{}
	for _, m := range methods {
		seen[m] = true
	}
	if !seen[MethodPassphrase] || !seen[MethodKeychain] {
		t.Fatalf("missing expected methods: %v", methods)
	}
}

func TestAddBadDEKLength(t *testing.T) {
	e := &Envelope{Version: envelopeVersion}
	short := []byte("too short")
	if err := e.AddPassphrase(short, "pp"); err == nil {
		t.Error("AddPassphrase should reject short dek")
	}
	wk, _ := GenerateKey()
	if err := e.AddRawKey(short, wk, MethodKeychain, "acct"); err == nil {
		t.Error("AddRawKey should reject short dek")
	}
}

func TestUnlockPassphraseNoMethod(t *testing.T) {
	// Build an envelope whose only method is keychain.
	e, dek, _ := NewEnvelopeWithPassphrase("pp")
	wk, _ := GenerateKey()
	_ = e.AddRawKey(dek, wk, MethodKeychain, "acct")
	if err := e.RemoveMethod(MethodPassphrase); err != nil {
		t.Fatal(err)
	}
	if _, err := e.UnlockWithPassphrase("pp"); err != ErrNoSuchMethod {
		t.Fatalf("expected ErrNoSuchMethod, got %v", err)
	}
}

func TestKeyRefMissing(t *testing.T) {
	e, _, _ := NewEnvelopeWithPassphrase("pp")
	if _, ok := e.KeyRef(MethodKeychain); ok {
		t.Error("KeyRef should report missing for absent method")
	}
}

func TestDecryptBlobBadKeyLength(t *testing.T) {
	blob, _ := EncryptBlob(mustKey(t), []byte("data"))
	if _, err := DecryptBlob([]byte("short"), blob); err == nil {
		t.Error("DecryptBlob should reject short key")
	}
}

func mustKey(t *testing.T) []byte {
	t.Helper()
	k, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}
