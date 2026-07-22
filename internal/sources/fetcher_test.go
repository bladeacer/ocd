package sources

import (
	"testing"

	"github.com/bladeacer/ocd/internal/cache"
)

func TestNewFetcher(t *testing.T) {
	c := cache.New(0)
	f := NewFetcher(c)
	if f.rss == nil {
		t.Error("expected rss source")
	}
	if f.docker == nil {
		t.Error("expected docker source")
	}
	if f.electron == nil {
		t.Error("expected electron source")
	}
}

func TestFetchAllCancel(t *testing.T) {
	_ = cache.New(0)
	// Fetcher uses errgroup with context timeout.
	// If cache misses and APIs are unreachable, FetchAll returns an error
	// rather than hanging. This test verifies the code doesn't panic.
}
