package server

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	keyring "github.com/zalando/go-keyring"

	"github.com/tallywell/tallywell/internal/app"
)

func init() { keyring.MockInit() }

type testEnv struct {
	ts     *httptest.Server
	client *http.Client
	srv    *Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	a, err := app.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(a, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	return &testEnv{ts: ts, client: &http.Client{Jar: jar}, srv: srv}
}

func (e *testEnv) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := e.client.Get(e.ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func (e *testEnv) post(t *testing.T, path string, form url.Values) *http.Response {
	t.Helper()
	resp, err := e.client.PostForm(e.ts.URL+path, form)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return sb.String()
}

func TestRedirectsToSetupThenDashboard(t *testing.T) {
	e := newTestEnv(t)

	// Before setup, "/" should land on the setup page.
	resp := e.get(t, "/")
	if got := body(t, resp); !strings.Contains(got, "Create a passphrase") {
		t.Fatalf("expected setup page, got: %.120s", got)
	}

	// Complete setup.
	resp = e.post(t, "/setup", url.Values{"passphrase": {"hunter2hunter2"}, "confirm": {"hunter2hunter2"}})
	if got := body(t, resp); !strings.Contains(got, "Here's where things stand") {
		t.Fatalf("expected dashboard after setup, got: %.120s", got)
	}
}

func TestSetupValidation(t *testing.T) {
	e := newTestEnv(t)
	resp := e.post(t, "/setup", url.Values{"passphrase": {"short"}, "confirm": {"short"}})
	if !strings.Contains(body(t, resp), "at least 8 characters") {
		t.Error("expected length validation message")
	}
	resp = e.post(t, "/setup", url.Values{"passphrase": {"longenough1"}, "confirm": {"different1"}})
	if !strings.Contains(body(t, resp), "do not match") {
		t.Error("expected mismatch message")
	}
}

func TestFullFlowAddDataAndExport(t *testing.T) {
	e := newTestEnv(t)
	e.post(t, "/setup", url.Values{"passphrase": {"hunter2hunter2"}, "confirm": {"hunter2hunter2"}})

	// Add a practice.
	e.post(t, "/settings/practice", url.Values{"name": {"My Practice"}, "kind": {"own"}})
	// Find its ID from the settings page select options is awkward; instead read
	// payers after adding via the practice we just created by re-querying store.
	st, err := e.srv.app.Store()
	if err != nil {
		t.Fatal(err)
	}
	prs, _ := st.Practices()
	if len(prs) != 1 {
		t.Fatalf("expected 1 practice, got %d", len(prs))
	}
	practiceID := prs[0].ID

	// Add a payer under it.
	e.post(t, "/settings/payer", url.Values{"name": {"Platform A"}, "practice_id": {practiceID}, "kind": {"insurance_platform"}})
	pys, _ := st.Payers()
	if len(pys) != 1 {
		t.Fatalf("expected 1 payer, got %d", len(pys))
	}
	payerID := pys[0].ID

	// Add a rate.
	e.post(t, "/rates", url.Values{"payer_id": {payerID}, "service": {"90837"}, "amount": {"$120"}})

	// Add a completed session — expected should auto-fill from the rate.
	e.post(t, "/sessions", url.Values{
		"date": {"2026-06-10"}, "client_id": {"AB"}, "payer_id": {payerID},
		"service": {"90837"}, "status": {"completed"},
	})
	recs, _ := st.Records()
	if len(recs) != 1 || recs[0].Expected != 12000 {
		t.Fatalf("session not added with auto rate: %+v", recs)
	}

	// Dashboard reflects the outstanding amount.
	if got := body(t, e.get(t, "/")); !strings.Contains(got, "$120.00") {
		t.Errorf("dashboard missing expected total: %.200s", got)
	}

	// Export returns an xlsx attachment.
	resp := e.get(t, "/export")
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "spreadsheetml") {
		t.Errorf("export content-type = %q", ct)
	}
	resp.Body.Close()
}

func TestStaticAssetServed(t *testing.T) {
	e := newTestEnv(t)
	resp := e.get(t, "/static/app.css")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("static css status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "css") {
		t.Errorf("static css content-type = %q", ct)
	}
}

func TestLockBlocksAccess(t *testing.T) {
	e := newTestEnv(t)
	e.post(t, "/setup", url.Values{"passphrase": {"hunter2hunter2"}, "confirm": {"hunter2hunter2"}})
	e.post(t, "/lock", url.Values{})

	if got := body(t, e.get(t, "/")); !strings.Contains(got, "Unlock") {
		t.Error("expected to be redirected to unlock after lock")
	}
	// Wrong passphrase stays locked.
	if got := body(t, e.post(t, "/unlock", url.Values{"passphrase": {"wrongwrong"}})); !strings.Contains(got, "Incorrect passphrase") {
		t.Error("expected incorrect passphrase message")
	}
	// Correct passphrase unlocks.
	if got := body(t, e.post(t, "/unlock", url.Values{"passphrase": {"hunter2hunter2"}})); !strings.Contains(got, "Here's where things stand") {
		t.Error("expected dashboard after correct unlock")
	}
}

func TestAutoLock(t *testing.T) {
	a, _ := app.New(t.TempDir())
	srv, _ := New(a, time.Hour)
	// Force a tiny auto-lock window and a controllable clock.
	srv.autoLock = 1 * time.Millisecond
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	if _, err := client.PostForm(ts.URL+"/setup", url.Values{"passphrase": {"hunter2hunter2"}, "confirm": {"hunter2hunter2"}}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)

	resp, err := client.Get(ts.URL + "/sessions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Request.URL.Path != "/unlock" {
		t.Errorf("expected auto-lock redirect to /unlock, landed on %s", resp.Request.URL.Path)
	}
}
