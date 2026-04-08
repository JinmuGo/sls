package hostinfo

import (
	"context"
)

// Fetcher implements finder.HostInfoFetcher using SSH probe.
type Fetcher struct {
	cache *Cache
}

// NewFetcher creates a new Fetcher with the given cache.
func NewFetcher(cache *Cache) *Fetcher {
	return &Fetcher{cache: cache}
}

// Get returns cached info for a host, or nil if not available or stale.
func (f *Fetcher) Get(host string) *HostInfo {
	if f.cache == nil {
		return nil
	}
	return f.cache.Get(host)
}

// FetchAsync fetches host info synchronously (called from a tea.Cmd goroutine).
func (f *Fetcher) FetchAsync(ctx context.Context, host string) (*HostInfo, error) {
	info := Fetch(ctx, host)
	return info, nil
}
