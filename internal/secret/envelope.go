package secret

import (
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Method identifies how a wrap's wrapping key is obtained.
type Method string

const (
	// MethodPassphrase derives the wrapping key from a user passphrase (Argon2id).
	MethodPassphrase Method = "passphrase"
	// MethodKeychain uses a random wrapping key stored in the OS keychain.
	MethodKeychain Method = "keychain"
	// MethodPasskey derives the wrapping key from a WebAuthn PRF secret (future).
	MethodPasskey Method = "passkey"
)

// envelopeVersion is the on-disk schema version for forward compatibility.
const envelopeVersion = 1

// Argon2id parameters. Tuned for an interactive unlock on a laptop.
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MiB
	argonThreads = 4
	saltLen      = 16
)

// ErrNoSuchMethod is returned when unlocking with a method that has no wrap.
var ErrNoSuchMethod = errors.New("secret: no wrap for that unlock method")

// ErrUnlock is returned when no wrap could recover the DEK with the given input.
var ErrUnlock = errors.New("secret: unable to unlock (wrong passphrase or key)")

// wrap is a single encrypted copy of the DEK plus the parameters needed to
// reconstruct its wrapping key.
type wrap struct {
	Method Method `json:"method"`

	// Passphrase-method KDF params.
	Salt    []byte `json:"salt,omitempty"`
	Time    uint32 `json:"time,omitempty"`
	Memory  uint32 `json:"memory,omitempty"`
	Threads uint8  `json:"threads,omitempty"`

	// KeyRef optionally names where a raw wrapping key lives (e.g. a keychain
	// account). The key itself is never stored in the envelope.
	KeyRef string `json:"key_ref,omitempty"`

	// Wrapped is the DEK encrypted under the wrapping key (nonce||ciphertext).
	Wrapped []byte `json:"wrapped"`
}

// Envelope holds the wrapped DEK under one or more unlock methods. It is safe
// to persist; it never contains the DEK or any wrapping key in the clear.
type Envelope struct {
	Version int    `json:"version"`
	Wraps   []wrap `json:"wraps"`
}

// deriveFromPassphrase computes an Argon2id wrapping key for the given params.
func deriveFromPassphrase(passphrase string, salt []byte, t, m uint32, p uint8) []byte {
	return argon2.IDKey([]byte(passphrase), salt, t, m, p, KeyLen)
}

// NewEnvelopeWithPassphrase generates a fresh DEK and returns an envelope whose
// first wrap is protected by passphrase. The DEK is returned so the caller can
// immediately encrypt data with it.
func NewEnvelopeWithPassphrase(passphrase string) (*Envelope, []byte, error) {
	dek, err := GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	e := &Envelope{Version: envelopeVersion}
	if err := e.AddPassphrase(dek, passphrase); err != nil {
		return nil, nil, err
	}
	return e, dek, nil
}

// AddPassphrase adds (or replaces) a passphrase wrap of dek.
func (e *Envelope) AddPassphrase(dek []byte, passphrase string) error {
	if len(dek) != KeyLen {
		return fmt.Errorf("secret: dek must be %d bytes", KeyLen)
	}
	salt, err := randBytes(saltLen)
	if err != nil {
		return err
	}
	wk := deriveFromPassphrase(passphrase, salt, argonTime, argonMemory, argonThreads)
	wrapped, err := seal(wk, dek)
	if err != nil {
		return err
	}
	e.replace(wrap{
		Method:  MethodPassphrase,
		Salt:    salt,
		Time:    argonTime,
		Memory:  argonMemory,
		Threads: argonThreads,
		Wrapped: wrapped,
	})
	return nil
}

// AddRawKey adds (or replaces) a wrap protected by a raw 32-byte wrapping key
// (used by the keychain tier; keyRef records where that key is stored).
func (e *Envelope) AddRawKey(dek, wrappingKey []byte, method Method, keyRef string) error {
	if len(dek) != KeyLen {
		return fmt.Errorf("secret: dek must be %d bytes", KeyLen)
	}
	wrapped, err := seal(wrappingKey, dek)
	if err != nil {
		return err
	}
	e.replace(wrap{Method: method, KeyRef: keyRef, Wrapped: wrapped})
	return nil
}

// replace inserts w, replacing any existing wrap of the same method.
func (e *Envelope) replace(w wrap) {
	for i := range e.Wraps {
		if e.Wraps[i].Method == w.Method {
			e.Wraps[i] = w
			return
		}
	}
	e.Wraps = append(e.Wraps, w)
}

// RemoveMethod removes the wrap for the given method, if present. It refuses to
// remove the last remaining wrap (which would make the DEK unrecoverable).
func (e *Envelope) RemoveMethod(method Method) error {
	idx := -1
	for i := range e.Wraps {
		if e.Wraps[i].Method == method {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrNoSuchMethod
	}
	if len(e.Wraps) == 1 {
		return errors.New("secret: cannot remove the only unlock method")
	}
	e.Wraps = append(e.Wraps[:idx], e.Wraps[idx+1:]...)
	return nil
}

// HasMethod reports whether the envelope has a wrap for method.
func (e *Envelope) HasMethod(method Method) bool {
	for i := range e.Wraps {
		if e.Wraps[i].Method == method {
			return true
		}
	}
	return false
}

// Methods returns the unlock methods currently present.
func (e *Envelope) Methods() []Method {
	out := make([]Method, 0, len(e.Wraps))
	for i := range e.Wraps {
		out = append(out, e.Wraps[i].Method)
	}
	return out
}

// UnlockWithPassphrase recovers the DEK using the passphrase wrap.
func (e *Envelope) UnlockWithPassphrase(passphrase string) ([]byte, error) {
	for i := range e.Wraps {
		w := e.Wraps[i]
		if w.Method != MethodPassphrase {
			continue
		}
		wk := deriveFromPassphrase(passphrase, w.Salt, w.Time, w.Memory, w.Threads)
		if dek, err := open(wk, w.Wrapped); err == nil {
			return dek, nil
		}
		return nil, ErrUnlock
	}
	return nil, ErrNoSuchMethod
}

// UnlockWithKey recovers the DEK using a raw wrapping key for the given method.
func (e *Envelope) UnlockWithKey(wrappingKey []byte, method Method) ([]byte, error) {
	for i := range e.Wraps {
		w := e.Wraps[i]
		if w.Method != method {
			continue
		}
		if dek, err := open(wrappingKey, w.Wrapped); err == nil {
			return dek, nil
		}
		return nil, ErrUnlock
	}
	return nil, ErrNoSuchMethod
}

// KeyRef returns the stored key reference for a method (e.g. keychain account).
func (e *Envelope) KeyRef(method Method) (string, bool) {
	for i := range e.Wraps {
		if e.Wraps[i].Method == method {
			return e.Wraps[i].KeyRef, true
		}
	}
	return "", false
}

// Marshal serializes the envelope to JSON for persistence.
func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalEnvelope parses an envelope previously produced by Marshal.
func UnmarshalEnvelope(data []byte) (*Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("secret: parse envelope: %w", err)
	}
	if e.Version != envelopeVersion {
		return nil, fmt.Errorf("secret: unsupported envelope version %d", e.Version)
	}
	return &e, nil
}
