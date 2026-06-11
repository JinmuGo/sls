package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTestCache writes a cache file under a temp HOME and returns nothing;
// callers set HOME via t.Setenv first.
func writeTestCache(t *testing.T, c cacheData) {
	t.Helper()
	p, err := cachePath()
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestCached(t *testing.T) {
	tests := []struct {
		name    string
		cache   *cacheData // nil = no cache file
		corrupt bool
		current string
		want    *Notice
	}{
		{"update available", &cacheData{LatestVersion: "1.1.2"}, false, "1.1.1", &Notice{Latest: "1.1.2"}},
		{"up to date", &cacheData{LatestVersion: "1.1.2"}, false, "1.1.2", nil},
		{"current newer than cache", &cacheData{LatestVersion: "1.1.2"}, false, "1.2.0", nil},
		{"v-prefixed tag in cache", &cacheData{LatestVersion: "v1.1.2"}, false, "1.1.1", &Notice{Latest: "v1.1.2"}},
		{"no cache file", nil, false, "1.1.1", nil},
		{"corrupt cache", nil, true, "1.1.1", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("SLS_NO_UPDATE_CHECK", "") // isolate from the developer's shell
			switch {
			case tt.corrupt:
				p, _ := cachePath()
				_ = os.MkdirAll(filepath.Dir(p), 0o755)
				_ = os.WriteFile(p, []byte("{not json"), 0o644)
			case tt.cache != nil:
				writeTestCache(t, *tt.cache)
			}

			got := Cached(tt.current)
			switch {
			case tt.want == nil && got != nil:
				t.Fatalf("Cached = %+v, want nil", got)
			case tt.want != nil && got == nil:
				t.Fatalf("Cached = nil, want %+v", tt.want)
			case tt.want != nil && got.Latest != tt.want.Latest:
				t.Fatalf("Cached.Latest = %q, want %q", got.Latest, tt.want.Latest)
			}
		})
	}
}

// TestCachedRespectsGuards verifies that the banner path honors the same opt-outs
// as the background refresh: a dev build and SLS_NO_UPDATE_CHECK must show no
// banner even when a newer version sits in the cache. A regression here is what
// made a locally-built `./sls` (version "dev") nag about an update.
func TestCachedRespectsGuards(t *testing.T) {
	t.Run("dev build shows no banner despite a newer cache", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("SLS_NO_UPDATE_CHECK", "")
		writeTestCache(t, cacheData{LatestVersion: "v1.3.0"})
		if got := Cached("dev"); got != nil {
			t.Errorf("Cached(%q) = %+v, want nil (dev builds must not nag)", "dev", got)
		}
	})

	t.Run("empty version shows no banner", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("SLS_NO_UPDATE_CHECK", "")
		writeTestCache(t, cacheData{LatestVersion: "v1.3.0"})
		if got := Cached(""); got != nil {
			t.Errorf("Cached(%q) = %+v, want nil", "", got)
		}
	})

	t.Run("opt-out env shows no banner", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("SLS_NO_UPDATE_CHECK", "1")
		writeTestCache(t, cacheData{LatestVersion: "v1.3.0"})
		if got := Cached("1.1.1"); got != nil {
			t.Errorf("Cached with SLS_NO_UPDATE_CHECK=1 = %+v, want nil", got)
		}
	})

	t.Run("normal build still shows the banner", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("SLS_NO_UPDATE_CHECK", "")
		writeTestCache(t, cacheData{LatestVersion: "v1.3.0"})
		if got := Cached("1.1.1"); got == nil {
			t.Error("Cached(\"1.1.1\") = nil, want a notice")
		}
	})
}

func TestFetchLatest(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    string
		wantErr bool
	}{
		{
			name: "happy path",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("User-Agent") == "" {
					t.Errorf("missing User-Agent header")
				}
				_, _ = w.Write([]byte(`{"tag_name":"v1.1.2"}`))
			},
			want: "v1.1.2",
		},
		{
			name: "rate limited",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			wantErr: true,
		},
		{
			name: "malformed json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{not json`))
			},
			wantErr: true,
		},
		{
			name: "empty tag",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"tag_name":""}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			orig := githubAPIBase
			githubAPIBase = srv.URL
			defer func() { githubAPIBase = orig }()

			got, err := fetchLatest("1.1.1")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("fetchLatest = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRefreshWritesCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.0"}`))
	}))
	defer srv.Close()
	orig := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = orig }()

	if err := refresh("1.1.1"); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	c, err := readCache()
	if err != nil || c == nil {
		t.Fatalf("readCache after refresh: c=%v err=%v", c, err)
	}
	if c.LatestVersion != "v1.2.0" {
		t.Errorf("cached latest = %q, want v1.2.0", c.LatestVersion)
	}
	if time.Since(c.LastChecked) > time.Minute {
		t.Errorf("LastChecked not recent: %v", c.LastChecked)
	}

	// And the banner should now reflect the freshly-written cache.
	if n := Cached("1.1.1"); n == nil || n.Latest != "v1.2.0" {
		t.Errorf("Cached after refresh = %+v, want Latest=v1.2.0", n)
	}
}

func TestCheckEnabled(t *testing.T) {
	tests := []struct {
		name        string
		current     string
		env         map[string]string
		interactive bool
		want        bool
	}{
		{"normal interactive", "1.1.1", nil, true, true},
		{"dev build", "dev", nil, true, false},
		{"empty version", "", nil, true, false},
		{"opt out with 1", "1.1.1", map[string]string{"SLS_NO_UPDATE_CHECK": "1"}, true, false},
		{"opt out with true", "1.1.1", map[string]string{"SLS_NO_UPDATE_CHECK": "true"}, true, false},
		{"not interactive", "1.1.1", nil, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkEnabled(tt.current, envFunc(tt.env), tt.interactive)
			if got != tt.want {
				t.Errorf("checkEnabled = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsRefresh(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	t.Run("disabled never refreshes", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if needsRefresh("dev", envFunc(nil), true, now) {
			t.Error("dev build should not refresh")
		}
	})

	t.Run("no cache refreshes", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if !needsRefresh("1.1.1", envFunc(nil), true, now) {
			t.Error("missing cache should trigger refresh")
		}
	})

	t.Run("fresh cache does not refresh", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		writeTestCache(t, cacheData{LastChecked: now.Add(-1 * time.Hour), LatestVersion: "1.1.1"})
		if needsRefresh("1.1.1", envFunc(nil), true, now) {
			t.Error("cache younger than TTL should not refresh")
		}
	})

	t.Run("stale cache refreshes", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		writeTestCache(t, cacheData{LastChecked: now.Add(-48 * time.Hour), LatestVersion: "1.1.1"})
		if !needsRefresh("1.1.1", envFunc(nil), true, now) {
			t.Error("cache older than TTL should refresh")
		}
	})
}

func TestOutdated(t *testing.T) {
	if !Outdated("1.1.1", "1.1.2") {
		t.Error("1.1.1 should be outdated vs 1.1.2")
	}
	if Outdated("1.1.2", "1.1.2") {
		t.Error("equal versions are not outdated")
	}
	if Outdated("1.2.0", "1.1.2") {
		t.Error("newer current is not outdated")
	}
}
