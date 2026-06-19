package secret

import (
	"bytes"
	"testing"
)

func TestPassphraseEnvelopeRoundTrip(t *testing.T) {
	e, dek, err := NewEnvelopeWithPassphrase("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if len(dek) != KeyLen {
		t.Fatalf("dek len = %d", len(dek))
	}

	got, err := e.UnlockWithPassphrase("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, dek) {
		t.Fatal("unlocked DEK does not match original")
	}
}

func TestWrongPassphraseFails(t *testing.T) {
	e, _, _ := NewEnvelopeWithPassphrase("right")
	if _, err := e.UnlockWithPassphrase("wrong"); err != ErrUnlock {
		t.Fatalf("expected ErrUnlock, got %v", err)
	}
}

func TestMarshalUnmarshalAndPersistUnlock(t *testing.T) {
	e, dek, _ := NewEnvelopeWithPassphrase("passphrase one")
	data, err := e.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, dek) {
		t.Fatal("serialized envelope leaked the DEK")
	}
	back, err := UnmarshalEnvelope(data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := back.UnlockWithPassphrase("passphrase one")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, dek) {
		t.Fatal("DEK mismatch after marshal round trip")
	}
}

func TestMultiWrapSameDEK(t *testing.T) {
	e, dek, _ := NewEnvelopeWithPassphrase("pp")

	// Add a keychain-style raw-key wrap of the SAME dek.
	wk, _ := GenerateKey()
	if err := e.AddRawKey(dek, wk, MethodKeychain, "acct-123"); err != nil {
		t.Fatal(err)
	}
	if !e.HasMethod(MethodKeychain) || !e.HasMethod(MethodPassphrase) {
		t.Fatal("expected both methods present")
	}
	if ref, ok := e.KeyRef(MethodKeychain); !ok || ref != "acct-123" {
		t.Fatalf("KeyRef = %q, %v", ref, ok)
	}

	// Both unlock paths must recover the identical DEK.
	viaPass, err := e.UnlockWithPassphrase("pp")
	if err != nil {
		t.Fatal(err)
	}
	viaKey, err := e.UnlockWithKey(wk, MethodKeychain)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(viaPass, dek) || !bytes.Equal(viaKey, dek) {
		t.Fatal("methods recovered different DEKs")
	}
}

func TestRemoveMethod(t *testing.T) {
	e, dek, _ := NewEnvelopeWithPassphrase("pp")
	wk, _ := GenerateKey()
	_ = e.AddRawKey(dek, wk, MethodKeychain, "acct")

	if err := e.RemoveMethod(MethodKeychain); err != nil {
		t.Fatal(err)
	}
	if e.HasMethod(MethodKeychain) {
		t.Fatal("keychain method should be gone")
	}
	// Removing the last remaining method must fail.
	if err := e.RemoveMethod(MethodPassphrase); err == nil {
		t.Fatal("expected error removing the only method")
	}
	// Removing a non-existent method returns ErrNoSuchMethod.
	if err := e.RemoveMethod(MethodPasskey); err != ErrNoSuchMethod {
		t.Fatalf("expected ErrNoSuchMethod, got %v", err)
	}
}

func TestUnlockMissingMethod(t *testing.T) {
	e, dek, _ := NewEnvelopeWithPassphrase("pp")
	if _, err := e.UnlockWithKey(dek, MethodKeychain); err != ErrNoSuchMethod {
		t.Fatalf("expected ErrNoSuchMethod, got %v", err)
	}
}

func TestUnmarshalRejectsBadVersion(t *testing.T) {
	if _, err := UnmarshalEnvelope([]byte(`{"version":99,"wraps":[]}`)); err == nil {
		t.Fatal("expected version error")
	}
	if _, err := UnmarshalEnvelope([]byte(`not json`)); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestReplacePassphraseRewrap(t *testing.T) {
	e, dek, _ := NewEnvelopeWithPassphrase("old")
	if err := e.AddPassphrase(dek, "new"); err != nil {
		t.Fatal(err)
	}
	if len(e.Wraps) != 1 {
		t.Fatalf("expected single passphrase wrap after rewrap, got %d", len(e.Wraps))
	}
	if _, err := e.UnlockWithPassphrase("old"); err != ErrUnlock {
		t.Fatal("old passphrase should no longer work")
	}
	got, err := e.UnlockWithPassphrase("new")
	if err != nil || !bytes.Equal(got, dek) {
		t.Fatalf("new passphrase should recover same dek: %v", err)
	}
}
