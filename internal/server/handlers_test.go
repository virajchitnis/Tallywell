package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tallywell/tallywell/internal/app"
)

// unlocked returns a testEnv with setup already completed.
func unlocked(t *testing.T) *testEnv {
	t.Helper()
	e := newTestEnv(t)
	e.post(t, "/setup", url.Values{"passphrase": {"hunter2hunter2"}, "confirm": {"hunter2hunter2"}})
	return e
}

// --- GET /setup ---

func TestSetupFormRendered(t *testing.T) {
	e := newTestEnv(t) // phase = NeedsSetup
	if got := body(t, e.get(t, "/setup")); !strings.Contains(got, "Create a passphrase") {
		t.Errorf("setup form not rendered: %.120s", got)
	}
}

func TestSetupFormRedirectsWhenAlreadySetUp(t *testing.T) {
	e := newTestEnv(t)
	e.post(t, "/setup", url.Values{"passphrase": {"hunter2hunter2"}, "confirm": {"hunter2hunter2"}})
	e.post(t, "/lock", url.Values{}) // phase = Locked

	resp := e.get(t, "/setup")
	if resp.Request.URL.Path != "/unlock" {
		t.Errorf("expected redirect to /unlock, got %s", resp.Request.URL.Path)
	}
}

// --- GET /unlock ---

func TestUnlockFormRedirectsToSetupWhenNeedsSetup(t *testing.T) {
	e := newTestEnv(t) // phase = NeedsSetup
	resp := e.get(t, "/unlock")
	if resp.Request.URL.Path != "/setup" {
		t.Errorf("expected redirect to /setup, got %s", resp.Request.URL.Path)
	}
}

func TestUnlockFormRedirectsToDashboardWhenAlreadyUnlocked(t *testing.T) {
	e := unlocked(t) // phase = Unlocked, valid session cookie in jar
	resp := e.get(t, "/unlock")
	if resp.Request.URL.Path != "/" {
		t.Errorf("expected redirect to /, got %s", resp.Request.URL.Path)
	}
}

func TestUnlockFormTimeoutFlash(t *testing.T) {
	e := newTestEnv(t)
	e.post(t, "/setup", url.Values{"passphrase": {"hunter2hunter2"}, "confirm": {"hunter2hunter2"}})
	e.post(t, "/lock", url.Values{})

	got := body(t, e.get(t, "/unlock?timeout=1"))
	if !strings.Contains(got, "inactivity") {
		t.Errorf("expected inactivity flash, got: %.120s", got)
	}
}

// --- SetQuitFunc ---

func TestSetQuitFunc(t *testing.T) {
	a, _ := app.New(t.TempDir())
	srv, _ := New(a, time.Hour, "dev")
	called := false
	srv.SetQuitFunc(func() { called = true })
	if srv.quit == nil {
		t.Fatal("quit func not set")
	}
	srv.quit()
	if !called {
		t.Error("quit func not invoked")
	}
}

// --- POST /quit ---

func TestQuitRendersPageAndCallsQuitFunc(t *testing.T) {
	e := unlocked(t)
	var calls atomic.Int32
	e.srv.SetQuitFunc(func() { calls.Add(1) })

	got := body(t, e.post(t, "/quit", url.Values{}))
	if !strings.Contains(got, "Tallywell") {
		t.Errorf("quit page body unexpected: %.120s", got)
	}

	// Quit func is called asynchronously after a short delay.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if calls.Load() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("quit func was not called within 2 seconds")
}

func TestQuitWithNoQuitFuncIsHarmless(t *testing.T) {
	e := unlocked(t)
	// No quit func registered — should not panic.
	resp := e.post(t, "/quit", url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// --- GET /import ---

func TestImportPageRendered(t *testing.T) {
	e := unlocked(t)
	got := body(t, e.get(t, "/import"))
	if !strings.Contains(got, "Import") {
		t.Errorf("import page not rendered: %.120s", got)
	}
}

// --- handleAddSession validation ---

func TestAddSessionBadDateRedirects(t *testing.T) {
	e := unlocked(t)
	resp := e.post(t, "/sessions", url.Values{
		"date": {"not-a-date"}, "client_id": {"AB"}, "payer_id": {"p1"},
		"service": {"90837"}, "status": {"completed"},
	})
	if resp.Request.URL.Path != "/sessions" {
		t.Errorf("expected redirect to /sessions on bad date, got %s", resp.Request.URL.Path)
	}
}

// --- handleAddRate validation ---

func TestAddRateBadAmountRedirects(t *testing.T) {
	e := unlocked(t)
	resp := e.post(t, "/rates", url.Values{
		"payer_id": {"py1"}, "service": {"90837"}, "amount": {"not-money"},
	})
	if resp.Request.URL.Path != "/rates" {
		t.Errorf("expected redirect to /rates on bad amount, got %s", resp.Request.URL.Path)
	}
}

func TestAddRateMissingPayerRedirects(t *testing.T) {
	e := unlocked(t)
	resp := e.post(t, "/rates", url.Values{
		"payer_id": {""}, "service": {"90837"}, "amount": {"$120"},
	})
	if resp.Request.URL.Path != "/rates" {
		t.Errorf("expected redirect to /rates on missing payer, got %s", resp.Request.URL.Path)
	}
}

// --- handleAddPractice validation ---

func TestAddPracticeMissingNameRedirects(t *testing.T) {
	e := unlocked(t)
	resp := e.post(t, "/settings/practice", url.Values{"name": {""}, "kind": {"own"}})
	if resp.Request.URL.Path != "/settings" {
		t.Errorf("expected redirect to /settings, got %s", resp.Request.URL.Path)
	}
}

func TestAddPracticeInvalidKindRedirects(t *testing.T) {
	e := unlocked(t)
	resp := e.post(t, "/settings/practice", url.Values{"name": {"My Practice"}, "kind": {"bogus"}})
	if resp.Request.URL.Path != "/settings" {
		t.Errorf("expected redirect to /settings, got %s", resp.Request.URL.Path)
	}
}

// --- handleAddPayer validation ---

func TestAddPayerMissingNameRedirects(t *testing.T) {
	e := unlocked(t)
	resp := e.post(t, "/settings/payer", url.Values{"name": {""}, "practice_id": {"pr1"}, "kind": {"insurance_platform"}})
	if resp.Request.URL.Path != "/settings" {
		t.Errorf("expected redirect to /settings, got %s", resp.Request.URL.Path)
	}
}

func TestAddPayerMissingPracticeRedirects(t *testing.T) {
	e := unlocked(t)
	resp := e.post(t, "/settings/payer", url.Values{"name": {"Alma"}, "practice_id": {""}, "kind": {"insurance_platform"}})
	if resp.Request.URL.Path != "/settings" {
		t.Errorf("expected redirect to /settings, got %s", resp.Request.URL.Path)
	}
}

// --- keychain auto-unlock and toggle ---

func TestKeychainAutoUnlock(t *testing.T) {
	e := unlocked(t)

	// Enable keychain auto-unlock via the settings toggle.
	resp := e.post(t, "/settings/keychain", url.Values{"action": {"add"}})
	if !strings.Contains(resp.Request.URL.String(), "keychain=ok") {
		t.Errorf("expected redirect to /settings?keychain=ok, got %s", resp.Request.URL)
	}

	// Lock the app to simulate a re-open.
	e.post(t, "/lock", url.Values{})

	// GET /unlock should auto-unlock via keychain and redirect to dashboard.
	resp = e.get(t, "/unlock")
	if resp.Request.URL.Path != "/" {
		t.Errorf("expected keychain auto-unlock to redirect to /, got %s", resp.Request.URL.Path)
	}
	if got := body(t, resp); !strings.Contains(got, "Here's where things stand") {
		t.Errorf("expected dashboard after keychain auto-unlock: %.120s", got)
	}
}

func TestKeychainDisable(t *testing.T) {
	e := unlocked(t)
	e.post(t, "/settings/keychain", url.Values{"action": {"add"}})

	// Disable.
	resp := e.post(t, "/settings/keychain", url.Values{"action": {"remove"}})
	if !strings.Contains(resp.Request.URL.String(), "keychain=ok") {
		t.Errorf("expected redirect to /settings?keychain=ok, got %s", resp.Request.URL)
	}

	// Lock — unlock page should show passphrase form, not auto-unlock.
	e.post(t, "/lock", url.Values{})
	resp = e.get(t, "/unlock")
	if resp.Request.URL.Path == "/" {
		t.Error("should not auto-unlock after disabling keychain")
	}
	if got := body(t, resp); !strings.Contains(got, "Unlock") {
		t.Errorf("expected unlock form after disable: %.120s", got)
	}
}

func TestSettingsShowsKeychainState(t *testing.T) {
	e := unlocked(t)

	// Before enabling: settings should offer "Enable auto-unlock".
	got := body(t, e.get(t, "/settings"))
	if !strings.Contains(got, "Enable auto-unlock") {
		t.Errorf("expected enable button before keychain enrolled: %.200s", got)
	}

	// After enabling: settings should offer "Disable auto-unlock".
	e.post(t, "/settings/keychain", url.Values{"action": {"add"}})
	got = body(t, e.get(t, "/settings"))
	if !strings.Contains(got, "Disable auto-unlock") {
		t.Errorf("expected disable button after keychain enrolled: %.200s", got)
	}
}

// --- reset ---

func TestResetFormRendered(t *testing.T) {
	e := unlocked(t)
	got := body(t, e.get(t, "/reset"))
	if !strings.Contains(got, "Reset all data") {
		t.Errorf("reset form not rendered: %.120s", got)
	}
	if !strings.Contains(got, "RESET") {
		t.Errorf("reset form missing confirmation prompt: %.120s", got)
	}
}

func TestResetWrongConfirmationShowsError(t *testing.T) {
	e := unlocked(t)
	got := body(t, e.post(t, "/reset", url.Values{"confirm": {"reset"}})) // lowercase
	if !strings.Contains(got, "capital letters") {
		t.Errorf("expected validation error, got: %.120s", got)
	}
	// App must still be unlocked — data not deleted.
	if got2 := body(t, e.get(t, "/")); !strings.Contains(got2, "Here's where things stand") {
		t.Errorf("app should still be unlocked after bad confirm: %.120s", got2)
	}
}

func TestResetCorrectConfirmationWipesDataAndRedirectsToSetup(t *testing.T) {
	e := newTestEnv(t)
	e.post(t, "/setup", url.Values{"passphrase": {"hunter2hunter2"}, "confirm": {"hunter2hunter2"}})

	// Add some data so we can confirm it's gone.
	e.post(t, "/settings/practice", url.Values{"name": {"My Practice"}, "kind": {"own"}})
	st, _ := e.srv.app.Store()
	prs, _ := st.Practices()
	if len(prs) != 1 {
		t.Fatalf("expected 1 practice before reset, got %d", len(prs))
	}

	resp := e.post(t, "/reset", url.Values{"confirm": {"RESET"}})
	if resp.Request.URL.Path != "/setup" {
		t.Errorf("expected redirect to /setup after reset, got %s", resp.Request.URL.Path)
	}
	if got := body(t, resp); !strings.Contains(got, "Create a passphrase") {
		t.Errorf("expected setup page after reset, got: %.120s", got)
	}

	// Session is cleared — guarded routes redirect to unlock/setup.
	if got := body(t, e.get(t, "/")); strings.Contains(got, "Here's where things stand") {
		t.Error("dashboard accessible after reset — session should have been cleared")
	}
}

// --- update check ---

func TestSemverLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v0.1.0", "v0.1.1", true},
		{"v0.1.1", "v0.1.0", false},
		{"v0.1.0", "v0.1.0", false},
		{"v0.9.0", "v0.10.0", true},
		{"v1.0.0", "v0.9.9", false},
		{"dev", "v0.1.0", true},
		{"v0.1.0", "dev", false},
	}
	for _, c := range cases {
		if got := semverLess(c.a, c.b); got != c.want {
			t.Errorf("semverLess(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestCheckUpdateCurrent(t *testing.T) {
	e := unlocked(t)
	e.srv.version = "v1.0.0"
	e.srv.updateChecker = func(_ context.Context, _ string) (string, error) { return "v1.0.0", nil }
	resp := e.post(t, "/settings/check-update", nil)
	if !strings.Contains(resp.Request.URL.RawQuery, "update=current") {
		t.Errorf("expected update=current in redirect, got %s", resp.Request.URL.RawQuery)
	}
}

func TestCheckUpdateAvailable(t *testing.T) {
	e := unlocked(t)
	e.srv.version = "v0.1.0"
	e.srv.updateChecker = func(_ context.Context, _ string) (string, error) { return "v0.2.0", nil }
	resp := e.post(t, "/settings/check-update", nil)
	q := resp.Request.URL.RawQuery
	if !strings.Contains(q, "update=available") || !strings.Contains(q, "latest=v0.2.0") {
		t.Errorf("expected update=available&latest=v0.2.0 in redirect, got %s", q)
	}
}

func TestCheckUpdateError(t *testing.T) {
	e := unlocked(t)
	e.srv.updateChecker = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("network failure")
	}
	resp := e.post(t, "/settings/check-update", nil)
	if !strings.Contains(resp.Request.URL.RawQuery, "update=error") {
		t.Errorf("expected update=error in redirect, got %s", resp.Request.URL.RawQuery)
	}
}

// --- guard: invalid session cookie ---

func TestGuardRejectsInvalidSession(t *testing.T) {
	e := unlocked(t)
	// Tamper with the session cookie.
	u, _ := url.Parse(e.ts.URL)
	for _, c := range e.client.Jar.Cookies(u) {
		if c.Name == sessionCookie {
			c.Value = "tampered"
			e.client.Jar.SetCookies(u, []*http.Cookie{c})
		}
	}
	resp := e.get(t, "/")
	if resp.Request.URL.Path != "/unlock" {
		t.Errorf("tampered session should redirect to /unlock, got %s", resp.Request.URL.Path)
	}
}
