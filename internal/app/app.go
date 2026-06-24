// Package app is Tallywell's application core: it ties the key envelope and the
// encrypted store into a setup / unlock / lock lifecycle, independent of the
// HTTP layer. The default (Protected) tier is implemented here; the envelope
// already supports additional unlock methods (keychain, passkey) for later.
package app

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	keyring "github.com/zalando/go-keyring"

	"github.com/tallywell/tallywell/internal/secret"
	"github.com/tallywell/tallywell/internal/store"
)

const (
	keychainService = "Tallywell"
	keychainAccount = "auto-unlock"
)

// keychainBackend is an injectable interface for OS keychain operations,
// allowing tests to substitute an in-memory mock.
type keychainBackend interface {
	Get(service, account string) (string, error)
	Set(service, account, value string) error
	Delete(service, account string) error
}

type realKeychain struct{}

func (realKeychain) Get(s, a string) (string, error)    { return keyring.Get(s, a) }
func (realKeychain) Set(s, a, v string) error           { return keyring.Set(s, a, v) }
func (realKeychain) Delete(s, a string) error           { return keyring.Delete(s, a) }

// Phase is the lifecycle state of the app.
type Phase int

const (
	// PhaseNeedsSetup means no envelope exists yet (first run).
	PhaseNeedsSetup Phase = iota
	// PhaseLocked means an envelope exists but the DEK is not in memory.
	PhaseLocked
	// PhaseUnlocked means the store is open and queryable.
	PhaseUnlocked
)

const (
	envelopeFile = "envelope.json"
	snapshotFile = "tracker.db.enc"
)

// ErrLocked is returned when an operation needs an unlocked store.
var ErrLocked = errors.New("app: locked")

// ErrAlreadySetup is returned when Setup is called but an envelope exists.
var ErrAlreadySetup = errors.New("app: already set up")

// App owns the persistent state and the in-memory unlocked state.
type App struct {
	dir      string
	keychain keychainBackend

	mu    sync.Mutex
	env   *secret.Envelope
	store *store.Store
	dek   []byte
}

// New creates an App rooted at dir, loading an existing envelope if present.
func New(dir string) (*App, error) {
	a := &App{dir: dir, keychain: realKeychain{}}
	data, err := os.ReadFile(a.envelopePath())
	switch {
	case errors.Is(err, os.ErrNotExist):
		// First run; nothing to load.
	case err != nil:
		return nil, err
	default:
		env, err := secret.UnmarshalEnvelope(data)
		if err != nil {
			return nil, err
		}
		a.env = env
	}
	return a, nil
}

func (a *App) envelopePath() string { return filepath.Join(a.dir, envelopeFile) }
func (a *App) snapshotPath() string { return filepath.Join(a.dir, snapshotFile) }

// Phase reports the current lifecycle state.
func (a *App) Phase() Phase {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch {
	case a.store != nil:
		return PhaseUnlocked
	case a.env != nil:
		return PhaseLocked
	default:
		return PhaseNeedsSetup
	}
}

// Setup creates a new passphrase-protected envelope and an empty store, leaving
// the app unlocked. It fails if an envelope already exists.
func (a *App) Setup(passphrase string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.env != nil {
		return ErrAlreadySetup
	}
	if err := os.MkdirAll(a.dir, 0o700); err != nil {
		return err
	}
	env, dek, err := secret.NewEnvelopeWithPassphrase(passphrase)
	if err != nil {
		return err
	}
	if err := a.writeEnvelope(env); err != nil {
		return err
	}
	st, err := store.Create(a.snapshotPath(), dek)
	if err != nil {
		return err
	}
	a.env, a.dek, a.store = env, dek, st
	return nil
}

// Unlock opens the store using the passphrase. It returns an error if the
// passphrase is wrong or there is nothing to unlock.
func (a *App) Unlock(passphrase string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.env == nil {
		return errors.New("app: not set up")
	}
	if a.store != nil {
		return nil // already unlocked
	}
	dek, err := a.env.UnlockWithPassphrase(passphrase)
	if err != nil {
		return err
	}
	st, err := store.Open(a.snapshotPath(), dek)
	if err != nil {
		return err
	}
	a.dek, a.store = dek, st
	return nil
}

// Lock closes the store and wipes the DEK from memory.
func (a *App) Lock() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lockLocked()
}

func (a *App) lockLocked() error {
	if a.store == nil {
		return nil
	}
	err := a.store.Close()
	a.store = nil
	for i := range a.dek {
		a.dek[i] = 0
	}
	a.dek = nil
	return err
}

// Store returns the open store, or ErrLocked if the app is not unlocked.
func (a *App) Store() (*store.Store, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store == nil {
		return nil, ErrLocked
	}
	return a.store, nil
}

// Dir returns the data directory for this app instance.
func (a *App) Dir() string { return a.dir }

// HasKeychainKey reports whether the envelope has a keychain-wrapped DEK.
func (a *App) HasKeychainKey() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.env != nil && a.env.HasMethod(secret.MethodKeychain)
}

// AddKeychainKey generates a random wrapping key, stores it in the OS keychain,
// and adds a keychain-wrapped copy of the DEK to the envelope. Requires unlocked.
func (a *App) AddKeychainKey() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store == nil {
		return ErrLocked
	}
	wk, err := secret.GenerateKey()
	if err != nil {
		return err
	}
	enc := base64.StdEncoding.EncodeToString(wk)
	if err := a.keychain.Set(keychainService, keychainAccount, enc); err != nil {
		return fmt.Errorf("app: store in keychain: %w", err)
	}
	if err := a.env.AddRawKey(a.dek, wk, secret.MethodKeychain, keychainAccount); err != nil {
		_ = a.keychain.Delete(keychainService, keychainAccount)
		return err
	}
	return a.writeEnvelope(a.env)
}

// RemoveKeychainKey removes the keychain entry and the keychain wrap from the
// envelope. It is a no-op if no keychain key is enrolled.
func (a *App) RemoveKeychainKey() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	_ = a.keychain.Delete(keychainService, keychainAccount)
	if err := a.env.RemoveMethod(secret.MethodKeychain); err != nil {
		if errors.Is(err, secret.ErrNoSuchMethod) {
			return nil
		}
		return err
	}
	return a.writeEnvelope(a.env)
}

// UnlockWithKeychain retrieves the wrapping key from the OS keychain and uses
// it to decrypt the DEK, leaving the app unlocked. Returns an error if no
// keychain key is enrolled or the keychain is unavailable.
func (a *App) UnlockWithKeychain() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.env == nil {
		return errors.New("app: not set up")
	}
	if a.store != nil {
		return nil // already unlocked
	}
	if !a.env.HasMethod(secret.MethodKeychain) {
		return errors.New("app: no keychain key enrolled")
	}
	enc, err := a.keychain.Get(keychainService, keychainAccount)
	if err != nil {
		return fmt.Errorf("app: keychain get: %w", err)
	}
	wk, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return fmt.Errorf("app: keychain decode: %w", err)
	}
	dek, err := a.env.UnlockWithKey(wk, secret.MethodKeychain)
	if err != nil {
		return err
	}
	st, err := store.Open(a.snapshotPath(), dek)
	if err != nil {
		return err
	}
	a.dek, a.store = dek, st
	return nil
}

// Reset permanently deletes all data and returns the app to the first-run state.
// The store is closed, the DEK is wiped from memory, and both the envelope and
// snapshot files are removed from disk. The caller should clear the session.
func (a *App) Reset() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store != nil {
		_ = a.store.Close()
		a.store = nil
	}
	for i := range a.dek {
		a.dek[i] = 0
	}
	a.dek = nil
	a.env = nil
	_ = os.Remove(a.snapshotPath())
	_ = os.Remove(a.envelopePath())
	return nil
}

// Backup writes an encrypted snapshot to path using the in-memory DEK.
func (a *App) Backup(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store == nil {
		return ErrLocked
	}
	return a.store.BackupTo(path, a.dek)
}

func (a *App) writeEnvelope(env *secret.Envelope) error {
	data, err := env.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(a.envelopePath(), data, 0o600)
}
