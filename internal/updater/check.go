// Package updater provides a lightweight, non-blocking update checker for sls.
//
// On launch, the dashboard reads a local cache (~/.config/sls/update.json) to
// decide whether to show an "update available" banner, while a background
// goroutine refreshes that cache from the GitHub Releases API for the next run.
// This mirrors the internal/pulse design: async, cached, and silent on failure.
//
// The check is on by default and can be disabled with SLS_NO_UPDATE_CHECK=1. It
// is skipped for dev builds and non-interactive sessions.
package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jinmugo/sls/internal/util"
)

const (
	cacheFile      = "update.json"
	checkTTL       = 24 * time.Hour
	requestTimeout = 5 * time.Second
	repoSlug       = "jinmugo/sls"
)

// githubAPIBase is overridable in tests to point at an httptest server.
var githubAPIBase = "https://api.github.com"

var httpClient = &http.Client{Timeout: requestTimeout}

// Notice is the information the dashboard banner needs about an available update.
type Notice struct {
	Latest string // the latest released version tag, e.g. "v1.1.2"
}

// cacheData is the on-disk shape of the update-check cache.
type cacheData struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version"`
}

// Cached returns a Notice if the cached latest version is newer than current,
// otherwise nil. A missing or corrupt cache yields nil (never an error) so the
// banner path is always silent.
func Cached(current string) *Notice {
	c, err := readCache()
	if err != nil || c == nil {
		return nil
	}
	if Outdated(current, c.LatestVersion) {
		return &Notice{Latest: c.LatestVersion}
	}
	return nil
}

// RefreshAsync refreshes the cache in the background when the check is enabled
// and the cache is missing or older than the TTL. It never blocks the caller and
// silently ignores all errors.
func RefreshAsync(current string) {
	if !needsRefresh(current, os.Getenv, isInteractive(), time.Now()) {
		return
	}
	go func() { _ = refresh(current) }()
}

// LatestVersion fetches the latest released version, writing it to the cache.
// On network failure it falls back to the cached value. The bool reports whether
// any version could be determined. Used by `sls update --check`.
func LatestVersion(current string) (string, bool) {
	if tag, err := fetchLatest(current); err == nil {
		_ = writeCache(&cacheData{LastChecked: time.Now().UTC(), LatestVersion: tag})
		return tag, true
	}
	if c, err := readCache(); err == nil && c != nil && c.LatestVersion != "" {
		return c.LatestVersion, true
	}
	return "", false
}

// Outdated reports whether latest is a newer version than current.
func Outdated(current, latest string) bool {
	return compareVersions(latest, current) > 0
}

// checkEnabled reports whether the update check should run at all.
func checkEnabled(current string, env func(string) string, interactive bool) bool {
	if v := env("SLS_NO_UPDATE_CHECK"); v == "1" || v == "true" {
		return false
	}
	if current == "" || current == "dev" {
		return false
	}
	return interactive
}

// needsRefresh reports whether a background refresh is warranted: the check must
// be enabled, and the cache must be missing, unreadable, or older than the TTL.
func needsRefresh(current string, env func(string) string, interactive bool, now time.Time) bool {
	if !checkEnabled(current, env, interactive) {
		return false
	}
	c, err := readCache()
	if err != nil || c == nil {
		return true
	}
	return now.Sub(c.LastChecked) >= checkTTL
}

// refresh fetches the latest version and writes it to the cache.
func refresh(current string) error {
	tag, err := fetchLatest(current)
	if err != nil {
		return err
	}
	return writeCache(&cacheData{LastChecked: time.Now().UTC(), LatestVersion: tag})
}

// fetchLatest queries the GitHub Releases API for the latest published tag.
func fetchLatest(current string) (string, error) {
	url := githubAPIBase + "/repos/" + repoSlug + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sls/"+current)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api: unexpected status %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.TagName == "" {
		return "", errors.New("github api: empty tag_name")
	}
	return payload.TagName, nil
}

// cachePath returns the path to the update-check cache file.
func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "sls", cacheFile), nil
}

// readCache reads and parses the cache file. A missing file returns an error
// (the standard not-exist error from os.ReadFile), so callers treat any error as
// "no usable cache".
func readCache() (*cacheData, error) {
	p, err := cachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c cacheData
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// writeCache atomically writes the cache file.
func writeCache(c *cacheData) error {
	p, err := cachePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return util.AtomicWriteFile(p, data, 0o644)
}

// isInteractive reports whether stdin is connected to a terminal.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
