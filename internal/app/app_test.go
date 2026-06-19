package app

import (
	"testing"

	"github.com/tallywell/tallywell/internal/model"
)

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
