package pulse

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// captured holds what the mock ingest endpoint received.
type captured struct {
	method string
	path   string
	auth   string
	ua     string
	body   otlpLogsBody
}

// newMockIngest returns a test server that records the first request into ch.
func newMockIngest(t *testing.T, ch chan<- captured) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var b otlpLogsBody
		_ = json.Unmarshal(raw, &b)
		ch <- captured{
			method: r.Method,
			path:   r.URL.Path,
			auth:   r.Header.Get("Authorization"),
			ua:     r.Header.Get("User-Agent"),
			body:   b,
		}
		w.WriteHeader(http.StatusOK)
	}))
}

// enableConsentIn writes an "enabled" consent file under home so hasConsent()
// returns true without an interactive prompt.
func enableConsentIn(t *testing.T, home string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "sls")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, consentFile), []byte("enabled\n"), 0o644); err != nil {
		t.Fatalf("write consent: %v", err)
	}
}

// attrValue returns the stringValue of the first attribute with the given key.
func attrValue(attrs []otlpKeyValue, key string) (string, bool) {
	for _, a := range attrs {
		if a.Key == key {
			return a.Value.StringValue, true
		}
	}
	return "", false
}

// TestTrackEmitsOTLP is the core wire-format guarantee: a tracked event reaches
// the ingest endpoint as one OTLP/HTTP log POST with the Bearer token, a named
// User-Agent, service.name="sls", and the flat attributes the Grafana panels
// require (event, anon_id, the command prop, and the os/arch/version context).
func TestTrackEmitsOTLP(t *testing.T) {
	home := t.TempDir()
	enableConsentIn(t, home)

	ch := make(chan captured, 1)
	srv := newMockIngest(t, ch)
	defer srv.Close()

	// Runtime overrides consumed by Init via getEnv. HOME redirects the consent +
	// anon_id files into the temp dir so the real ~/.config/sls is never touched.
	t.Setenv("HOME", home)
	t.Setenv("PULSE_DISABLED", "")
	t.Setenv("SLS_TELEMETRY", "")
	t.Setenv("TELEMETRY_ENDPOINT", srv.URL)
	t.Setenv("TELEMETRY_API_KEY", "test-cli-token")

	// Reset package state so the test is independent of init order.
	disabled, started = false, false

	Init("9.9.9")
	if disabled || !started {
		t.Fatalf("Init did not enable telemetry (disabled=%v started=%v)", disabled, started)
	}
	Track("command_run", Props{"command": "connect"})
	Shutdown() // synchronous flush

	var got captured
	select {
	case got = <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("ingest endpoint received no request")
	}

	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if got.path != "/v1/logs" {
		t.Errorf("path = %q, want /v1/logs", got.path)
	}
	if got.auth != "Bearer test-cli-token" {
		t.Errorf("Authorization = %q, want Bearer test-cli-token", got.auth)
	}
	if got.ua != "sls/9.9.9" {
		t.Errorf("User-Agent = %q, want sls/9.9.9", got.ua)
	}

	if len(got.body.ResourceLogs) != 1 {
		t.Fatalf("resourceLogs len = %d, want 1", len(got.body.ResourceLogs))
	}
	rl := got.body.ResourceLogs[0]
	if v, _ := attrValue(rl.Resource.Attributes, "service.name"); v != serviceName {
		t.Errorf("resource service.name = %q, want %q", v, serviceName)
	}
	if len(rl.ScopeLogs) != 1 || len(rl.ScopeLogs[0].LogRecords) != 1 {
		t.Fatalf("want exactly one log record, got scopeLogs=%d", len(rl.ScopeLogs))
	}
	rec := rl.ScopeLogs[0].LogRecords[0]
	if rec.Body.StringValue != "command_run" {
		t.Errorf("record body = %q, want command_run", rec.Body.StringValue)
	}
	if rec.TimeUnixNano == "" {
		t.Error("record timeUnixNano is empty")
	}
	if v, _ := attrValue(rec.Attributes, "event"); v != "command_run" {
		t.Errorf("attr event = %q, want command_run", v)
	}
	if v, ok := attrValue(rec.Attributes, "anon_id"); !ok || v == "" {
		t.Errorf("attr anon_id missing/empty (ok=%v val=%q)", ok, v)
	}
	if v, _ := attrValue(rec.Attributes, "command"); v != "connect" {
		t.Errorf("attr command = %q, want connect", v)
	}
	if v, _ := attrValue(rec.Attributes, "version"); v != "9.9.9" {
		t.Errorf("attr version = %q, want 9.9.9", v)
	}
}

// TestDisabledViaEnv verifies the opt-out env vars keep telemetry fully off even
// when consent is on: no request must reach the endpoint.
func TestDisabledViaEnv(t *testing.T) {
	home := t.TempDir()
	enableConsentIn(t, home)

	ch := make(chan captured, 1)
	srv := newMockIngest(t, ch)
	defer srv.Close()

	t.Setenv("HOME", home)
	t.Setenv("TELEMETRY_ENDPOINT", srv.URL)
	t.Setenv("TELEMETRY_API_KEY", "test-cli-token")
	t.Setenv("PULSE_DISABLED", "1")

	disabled, started = false, false

	Init("9.9.9")
	if !disabled {
		t.Fatal("PULSE_DISABLED=1 should disable telemetry")
	}
	Track("command_run", Props{"command": "connect"})
	Shutdown()

	select {
	case <-ch:
		t.Fatal("disabled telemetry still sent a request")
	case <-time.After(300 * time.Millisecond):
		// expected: nothing sent
	}
}

// TestAnonIDStable verifies the anon_id persists across Init calls (same install
// -> same id) so count_uniq(anon_id) counts installs, not invocations.
func TestAnonIDStable(t *testing.T) {
	home := t.TempDir()

	first := func() string {
		ch := make(chan captured, 1)
		srv := newMockIngest(t, ch)
		defer srv.Close()
		enableConsentIn(t, home)
		t.Setenv("HOME", home)
		t.Setenv("PULSE_DISABLED", "")
		t.Setenv("SLS_TELEMETRY", "")
		t.Setenv("TELEMETRY_ENDPOINT", srv.URL)
		t.Setenv("TELEMETRY_API_KEY", "tok")
		disabled, started = false, false
		Init("1.0.0")
		Track("command_run", Props{"command": "scan"})
		Shutdown()
		got := <-ch
		v, _ := attrValue(got.body.ResourceLogs[0].ScopeLogs[0].LogRecords[0].Attributes, "anon_id")
		return v
	}

	a := first()
	b := first()
	if a == "" || a != b {
		t.Errorf("anon_id not stable across runs: %q vs %q", a, b)
	}
}
