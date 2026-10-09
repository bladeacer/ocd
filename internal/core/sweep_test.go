package core

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// asarServer serves a valid ASAR bundle for every version in ok, a 404 for
// every version in missing, and a 500 for every other version.
func asarServer(t *testing.T, ok, missing map[string]bool) (*httptest.Server, func()) {
	t.Helper()
	payload := buildMinimalASARGz(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := r.URL.Query().Get("v")
		if version == "" {
			version = versionFromPath(r.URL.Path)
		}
		if missing[version] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if !ok[version] {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(payload)
	}))

	origURL := asarReleaseURL
	origClient := httpClient
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz?v=%s"
	httpClient = ts.Client()

	return ts, func() {
		asarReleaseURL = origURL
		httpClient = origClient
	}
}

func versionFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimPrefix(parts[1], "v")
}

func useTempCSSDir(t *testing.T) {
	t.Helper()
	orig := CSSDir
	CSSDir = t.TempDir()
	t.Cleanup(func() { CSSDir = orig })
}

func TestEnsureAllCSSDownloadsEverything(t *testing.T) {
	useTempCSSDir(t)
	versions := map[string]bool{"1.0.0": true, "1.1.0": true, "1.2.0": true}
	ts, cleanup := asarServer(t, versions, nil)
	defer cleanup()
	defer ts.Close()

	res := EnsureAllCSS(SweepOptions{Versions: []string{"1.2.0", "1.0.0", "1.1.0"}})

	if res.Requested != 3 {
		t.Errorf("Requested = %d, want 3", res.Requested)
	}
	if res.Downloaded != 3 {
		t.Errorf("Downloaded = %d, want 3", res.Downloaded)
	}
	if res.Cached != 0 || res.Failed != 0 || res.Unavailable != 0 {
		t.Errorf("unexpected counts: %+v", res)
	}
	for v := range versions {
		if !CSSCached(v) {
			t.Errorf("expected %s to be cached", v)
		}
	}
	if got := res.String(); !strings.Contains(got, "3 downloaded") {
		t.Errorf("String() = %q, want it to mention 3 downloaded", got)
	}
}

func TestEnsureAllCSSSkipsFreshCache(t *testing.T) {
	useTempCSSDir(t)
	var mu sync.Mutex
	hits := 0
	payload := buildMinimalASARGz(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		_, _ = w.Write(payload)
	}))
	defer ts.Close()

	origURL := asarReleaseURL
	origClient := httpClient
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	httpClient = ts.Client()
	defer func() { asarReleaseURL = origURL; httpClient = origClient }()

	res := EnsureAllCSS(SweepOptions{Versions: []string{"1.0.0"}})

	mu.Lock()
	first := hits
	mu.Unlock()
	if first != 1 {
		t.Fatalf("expected 1 download on first sweep, got %d", first)
	}
	if res.Downloaded != 1 {
		t.Fatalf("Downloaded = %d, want 1", res.Downloaded)
	}

	// A second sweep with a fresh entry must not touch the network.
	res = EnsureAllCSS(SweepOptions{Versions: []string{"1.0.0"}, CacheDays: DefaultCacheDays})

	mu.Lock()
	second := hits
	mu.Unlock()
	if second != 1 {
		t.Errorf("expected no extra download, got %d total", second)
	}
	if res.Cached != 1 || res.Downloaded != 0 {
		t.Errorf("unexpected counts: %+v", res)
	}
}

func TestEnsureAllCSSExpiredEntryIsDownloadedAgain(t *testing.T) {
	useTempCSSDir(t)
	var mu sync.Mutex
	hits := 0
	payload := buildMinimalASARGz(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		_, _ = w.Write(payload)
	}))
	defer ts.Close()

	origURL := asarReleaseURL
	origClient := httpClient
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	httpClient = ts.Client()
	defer func() { asarReleaseURL = origURL; httpClient = origClient }()

	EnsureAllCSS(SweepOptions{Versions: []string{"1.0.0"}})

	// Age the cached file past the cache window.
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(CSSPath("1.0.0"), old, old); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	if CSSFresh("1.0.0", 14*24*time.Hour) {
		t.Fatal("entry should be stale after 30 days with a 14 day window")
	}

	res := EnsureAllCSS(SweepOptions{Versions: []string{"1.0.0"}, CacheDays: 14})

	mu.Lock()
	total := hits
	mu.Unlock()
	if total != 2 {
		t.Errorf("expected a second download, got %d total", total)
	}
	if res.Downloaded != 1 {
		t.Errorf("Downloaded = %d, want 1", res.Downloaded)
	}
}

func TestEnsureAllCSSForceReplacesFreshEntry(t *testing.T) {
	useTempCSSDir(t)
	ts, cleanup := asarServer(t, map[string]bool{"1.0.0": true}, nil)
	defer cleanup()
	defer ts.Close()

	EnsureAllCSS(SweepOptions{Versions: []string{"1.0.0"}})

	res := EnsureAllCSS(SweepOptions{Versions: []string{"1.0.0"}, Force: true})
	if res.Downloaded != 1 {
		t.Errorf("Downloaded = %d, want 1 with Force", res.Downloaded)
	}
	if res.Cached != 0 {
		t.Errorf("Cached = %d, want 0 with Force", res.Cached)
	}
}

func TestEnsureAllCSSMissingReleaseIsUnavailable(t *testing.T) {
	useTempCSSDir(t)
	ts, cleanup := asarServer(t,
		map[string]bool{"1.0.0": true},
		map[string]bool{"0.1.0": true, "0.2.0": true},
	)
	defer cleanup()
	defer ts.Close()

	res := EnsureAllCSS(SweepOptions{
		Versions: []string{"0.2.0", "1.0.0", "0.1.0"},
		Log:      &bytes.Buffer{},
	})

	if res.Unavailable != 2 {
		t.Errorf("Unavailable = %d, want 2", res.Unavailable)
	}
	if res.Failed != 0 {
		t.Errorf("Failed = %d, want 0 (a missing release is not a failure)", res.Failed)
	}
	if res.Downloaded != 1 {
		t.Errorf("Downloaded = %d, want 1", res.Downloaded)
	}
	want := []string{"0.1.0", "0.2.0"}
	if len(res.UnavailableVersions) != 2 || res.UnavailableVersions[0] != want[0] || res.UnavailableVersions[1] != want[1] {
		t.Errorf("UnavailableVersions = %v, want %v", res.UnavailableVersions, want)
	}
	if CSSCached("0.1.0") {
		t.Error("a version with no release must not be cached")
	}
}

func TestEnsureAllCSSServerErrorIsFailure(t *testing.T) {
	useTempCSSDir(t)
	ts, cleanup := asarServer(t, map[string]bool{}, map[string]bool{})
	defer cleanup()
	defer ts.Close()

	res := EnsureAllCSS(SweepOptions{Versions: []string{"1.0.0"}, Log: &bytes.Buffer{}})

	if res.Failed != 1 {
		t.Errorf("Failed = %d, want 1", res.Failed)
	}
	if len(res.FailedVersions) != 1 || res.FailedVersions[0] != "1.0.0" {
		t.Errorf("FailedVersions = %v, want [1.0.0]", res.FailedVersions)
	}
}

func TestEnsureAllCSSEmptyVersions(t *testing.T) {
	useTempCSSDir(t)
	res := EnsureAllCSS(SweepOptions{})
	if res == nil {
		t.Fatal("EnsureAllCSS must never return nil")
	}
	if res.Requested != 0 {
		t.Errorf("Requested = %d, want 0", res.Requested)
	}
}

func TestEnsureAllCSSProgressRunsOncePerVersion(t *testing.T) {
	useTempCSSDir(t)
	ts, cleanup := asarServer(t, map[string]bool{"1.0.0": true, "1.1.0": true, "1.2.0": true}, nil)
	defer cleanup()
	defer ts.Close()

	var mu sync.Mutex
	seen := map[int]int{}

	res := EnsureAllCSS(SweepOptions{
		Versions:    []string{"1.0.0", "1.1.0", "1.2.0"},
		Concurrency: 1,
		Progress: func(done, total int, version string, status SweepStatus, err error) {
			mu.Lock()
			defer mu.Unlock()
			if total != 3 {
				t.Errorf("total = %d, want 3", total)
			}
			if err != nil {
				t.Errorf("unexpected error for %s: %v", version, err)
			}
			seen[done]++
		},
	})

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 3 {
		t.Errorf("progress fired %d times, want 3", len(seen))
	}
	if res.Requested != 3 {
		t.Errorf("Requested = %d, want 3", res.Requested)
	}
}

func TestEnsureAllCSSConcurrencyOne(t *testing.T) {
	useTempCSSDir(t)
	payload := buildMinimalASARGz(t)
	var mu sync.Mutex
	inFlight, maxInFlight := 0, 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()

		time.Sleep(5 * time.Millisecond)

		mu.Lock()
		inFlight--
		mu.Unlock()
		_, _ = w.Write(payload)
	}))
	defer ts.Close()

	origURL := asarReleaseURL
	origClient := httpClient
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	httpClient = ts.Client()
	defer func() { asarReleaseURL = origURL; httpClient = origClient }()

	versions := []string{"1.0.0", "1.0.1", "1.0.2", "1.0.3"}
	res := EnsureAllCSS(SweepOptions{Versions: versions, Concurrency: 1})

	if res.Downloaded != len(versions) {
		t.Errorf("Downloaded = %d, want %d", res.Downloaded, len(versions))
	}
	mu.Lock()
	defer mu.Unlock()
	if maxInFlight > 1 {
		t.Errorf("maxInFlight = %d, want at most 1 when Concurrency is 1", maxInFlight)
	}
}

func TestEnsureAllCSSKeepsExistingFileWhenRefreshFails(t *testing.T) {
	useTempCSSDir(t)
	payload := buildMinimalASARGz(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	ts.Close()

	origURL := asarReleaseURL
	origClient := httpClient
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	httpClient = ts.Client()
	defer func() { asarReleaseURL = origURL; httpClient = origClient }()

	// Seed the cache by hand, then point at a server that is down.
	dir := CSSPath("1.0.0")
	if err := os.MkdirAll(strings.TrimSuffix(dir, "/app.css"), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(CSSPath("1.0.0"), []byte("cached{}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	res := EnsureAllCSS(SweepOptions{Versions: []string{"1.0.0"}, Force: true})
	if res.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", res.Failed)
	}

	data, err := os.ReadFile(CSSPath("1.0.0"))
	if err != nil {
		t.Fatalf("cached file was removed: %v", err)
	}
	if string(data) != "cached{}" {
		t.Errorf("cached file was overwritten: %q", string(data))
	}
}

func TestSweepOptionsTTL(t *testing.T) {
	if got := (SweepOptions{}).TTL(); got != 0 {
		t.Errorf("TTL with no cache days = %v, want 0", got)
	}
	if got := (SweepOptions{CacheDays: -1}).TTL(); got != 0 {
		t.Errorf("TTL with negative cache days = %v, want 0", got)
	}
	want := 3 * 24 * time.Hour
	if got := (SweepOptions{CacheDays: 3}).TTL(); got != want {
		t.Errorf("TTL = %v, want %v", got, want)
	}
}

func TestDefaultCacheDays(t *testing.T) {
	if DefaultCacheDays != 14 {
		t.Errorf("DefaultCacheDays = %d, want 14", DefaultCacheDays)
	}
}

func TestNoReleaseErrorIsMatchable(t *testing.T) {
	useTempCSSDir(t)
	ts, cleanup := asarServer(t, map[string]bool{}, map[string]bool{"9.9.9": true})
	defer cleanup()
	defer ts.Close()

	_, err := ExtractCSS("9.9.9")
	if !errors.Is(err, ErrNoRelease) {
		t.Errorf("errors.Is(err, ErrNoRelease) = false for %v", err)
	}
}
