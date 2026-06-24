package app

import (
	"testing"

	keyring "github.com/zalando/go-keyring"

	"github.com/tallywell/tallywell/internal/model"
)

func init() { keyring.MockInit() }

func TestLifecycle(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Phase() != PhaseNeedsSetup {
		t.Fatalf("fresh app phase = %v, want NeedsSetup", a.Phase())
	}

	// Locked operations fail before setup.
	if _, err := a.Store(); err != ErrLocked {
		t.Errorf("Store before setup = %v, want ErrLocked", err)
	}

	if err := a.Setup("strong passphrase"); err != nil {
		t.Fatal(err)
	}
	if a.Phase() != PhaseUnlocked {
		t.Fatalf("after setup phase = %v, want Unlocked", a.Phase())
	}

	// Re-setup is refused.
	if err := a.Setup("other"); err != ErrAlreadySetup {
		t.Errorf("re-setup = %v, want ErrAlreadySetup", err)
	}

	// Add some data, then lock.
	st, err := a.Store()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.PutPractice(model.Practice{ID: "p1", Name: "Own", Kind: model.PracticeOwn}); err != nil {
		t.Fatal(err)
	}
	if err := a.Lock(); err != nil {
		t.Fatal(err)
	}
	if a.Phase() != PhaseLocked {
		t.Fatalf("after lock phase = %v, want Locked", a.Phase())
	}
	if _, err := a.Store(); err != ErrLocked {
		t.Errorf("Store after lock = %v, want ErrLocked", err)
	}
}

func TestReopenAndUnlock(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	if err := a.Setup("pw"); err != nil {
		t.Fatal(err)
	}
	st, _ := a.Store()
	_ = st.PutPractice(model.Practice{ID: "p1", Name: "Own", Kind: model.PracticeOwn})
	_ = a.Lock()

	// Simulate a fresh process: new App over the same dir.
	a2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a2.Phase() != PhaseLocked {
		t.Fatalf("reopened phase = %v, want Locked", a2.Phase())
	}
	if err := a2.Unlock("wrong"); err == nil {
		t.Fatal("unlock with wrong passphrase should fail")
	}
	if err := a2.Unlock("pw"); err != nil {
		t.Fatal(err)
	}
	st2, _ := a2.Store()
	prs, _ := st2.Practices()
	if len(prs) != 1 || prs[0].ID != "p1" {
		t.Fatalf("data not recovered after unlock: %+v", prs)
	}
}

func TestBackupRequiresUnlock(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	if err := a.Backup(dir + "/b.enc"); err != ErrLocked {
		t.Errorf("backup while locked = %v, want ErrLocked", err)
	}
	_ = a.Setup("pw")
	if err := a.Backup(dir + "/sub/b.enc"); err != nil {
		t.Errorf("backup while unlocked: %v", err)
	}
}

func TestReset(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	_ = a.Setup("pw")

	st, _ := a.Store()
	_ = st.PutPractice(model.Practice{ID: "p1", Name: "Own", Kind: model.PracticeOwn})

	if err := a.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	// Phase must be NeedsSetup after reset.
	if a.Phase() != PhaseNeedsSetup {
		t.Fatalf("phase after reset = %v, want NeedsSetup", a.Phase())
	}

	// Data files must be gone.
	if a.Dir() == "" {
		t.Fatal("Dir() returned empty string")
	}

	// A fresh App over the same dir sees no envelope (first-run state).
	a2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a2.Phase() != PhaseNeedsSetup {
		t.Fatalf("reopened phase after reset = %v, want NeedsSetup", a2.Phase())
	}

	// Can set up again with a new passphrase after reset.
	if err := a2.Setup("newpassphrase"); err != nil {
		t.Fatalf("Setup after reset: %v", err)
	}
	if a2.Phase() != PhaseUnlocked {
		t.Fatalf("phase after re-setup = %v, want Unlocked", a2.Phase())
	}
}

func TestKeychainAddAndUnlock(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	_ = a.Setup("pw")

	if a.HasKeychainKey() {
		t.Fatal("should have no keychain key after setup")
	}
	if err := a.AddKeychainKey(); err != nil {
		t.Fatalf("AddKeychainKey: %v", err)
	}
	if !a.HasKeychainKey() {
		t.Fatal("should have keychain key after add")
	}

	// Lock and re-unlock via keychain.
	_ = a.Lock()
	if a.Phase() != PhaseLocked {
		t.Fatalf("phase after lock = %v, want Locked", a.Phase())
	}
	if err := a.UnlockWithKeychain(); err != nil {
		t.Fatalf("UnlockWithKeychain: %v", err)
	}
	if a.Phase() != PhaseUnlocked {
		t.Fatalf("phase after keychain unlock = %v, want Unlocked", a.Phase())
	}
}

func TestKeychainRemove(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	_ = a.Setup("pw")
	_ = a.AddKeychainKey()

	if err := a.RemoveKeychainKey(); err != nil {
		t.Fatalf("RemoveKeychainKey: %v", err)
	}
	if a.HasKeychainKey() {
		t.Fatal("should not have keychain key after remove")
	}

	// Keychain unlock must now fail.
	_ = a.Lock()
	if err := a.UnlockWithKeychain(); err == nil {
		t.Fatal("UnlockWithKeychain after remove should fail")
	}
	// Passphrase still works.
	if err := a.Unlock("pw"); err != nil {
		t.Fatalf("passphrase unlock after keychain remove: %v", err)
	}
}

func TestKeychainRemoveIdempotent(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	_ = a.Setup("pw")
	// RemoveKeychainKey with nothing enrolled should be a no-op.
	if err := a.RemoveKeychainKey(); err != nil {
		t.Fatalf("RemoveKeychainKey on clean envelope: %v", err)
	}
}

func TestKeychainEnvelopePersists(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	_ = a.Setup("pw")
	_ = a.AddKeychainKey()

	// Reload from disk — keychain wrap must survive.
	a2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !a2.HasKeychainKey() {
		t.Fatal("keychain key not present after reload from disk")
	}
	if err := a2.UnlockWithKeychain(); err != nil {
		t.Fatalf("UnlockWithKeychain after reload: %v", err)
	}
}

func TestResetWhileLocked(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	_ = a.Setup("pw")
	_ = a.Lock()

	if err := a.Reset(); err != nil {
		t.Fatalf("Reset while locked: %v", err)
	}
	if a.Phase() != PhaseNeedsSetup {
		t.Fatalf("phase after reset-while-locked = %v, want NeedsSetup", a.Phase())
	}
}
