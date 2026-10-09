package core

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// DefaultCacheDays is the number of days a cached app.css stays fresh when
// the user does not set a value.
const DefaultCacheDays = 14

// DefaultSweepConcurrency is the number of ASAR bundles downloaded at once
// during a sweep. The value keeps the load on GitHub modest while still
// finishing in a sensible time.
const DefaultSweepConcurrency = 8

// SweepStatus describes what EnsureAllCSS did with a single version.
type SweepStatus string

const (
	// SweepCached means a fresh cached copy was present and left alone.
	SweepCached SweepStatus = "cached"
	// SweepDownloaded means app.css was downloaded during this sweep.
	SweepDownloaded SweepStatus = "downloaded"
	// SweepUnavailable means GitHub has no release for the version.
	SweepUnavailable SweepStatus = "unavailable"
	// SweepFailed means the download failed for a reason other than a
	// missing release, such as a network error.
	SweepFailed SweepStatus = "failed"
)

// SweepOptions controls an EnsureAllCSS run.
type SweepOptions struct {
	// Versions lists the version labels to make sure are cached.
	Versions []string
	// CacheDays is how many days a cached entry stays fresh. Zero or less
	// means entries never expire, so every missing entry is downloaded.
	CacheDays int
	// Force downloads every version, even when a fresh entry exists.
	Force bool
	// Concurrency caps the number of parallel downloads. Values of zero or
	// less fall back to DefaultSweepConcurrency.
	Concurrency int
	// Progress, when set, is called once at the end of the run with the
	// result, so a caller can report without writing its own counters.
	Progress func(*SweepResult)
	// Log, when set, receives one line per version. It is a convenience
	// wrapper for callers that only want text output.
	Log io.Writer
}

// SweepResult summarises an EnsureAllCSS run.
type SweepResult struct {
	// Requested is the number of versions the sweep was asked to cover.
	Requested int
	// Cached counts versions that were already present and fresh.
	Cached int
	// Downloaded counts versions fetched during this sweep.
	Downloaded int
	// Unavailable counts versions with no ASAR release on GitHub.
	Unavailable int
	// Failed counts versions whose download failed.
	Failed int
	// DownloadedVersions lists the versions fetched, in ascending order.
	DownloadedVersions []string
	// UnavailableVersions lists the versions with no release.
	UnavailableVersions []string
	// FailedVersions lists the versions whose download failed.
	FailedVersions []string
}

// String renders a one line summary for the terminal.
func (r *SweepResult) String() string {
	return fmt.Sprintf("%d versions: %d cached, %d downloaded, %d unavailable, %d failed",
		r.Requested, r.Cached, r.Downloaded, r.Unavailable, r.Failed)
}

// TTL converts CacheDays into a duration for the freshness checks.
func (o SweepOptions) TTL() time.Duration {
	if o.CacheDays <= 0 {
		return 0
	}
	return time.Duration(o.CacheDays) * 24 * time.Hour
}

// ParallelFor runs fn for every index in [0,n) across at most limit
// goroutines and waits for all of them. A limit of zero or less falls back to
// DefaultSweepConcurrency.
//
// The indices are independent, so fn must not rely on the order they run in
// and must do its own locking when it writes shared state.
func ParallelFor(n, limit int, fn func(i int)) {
	if n <= 0 {
		return
	}
	if limit <= 0 {
		limit = DefaultSweepConcurrency
	}
	if limit > n {
		limit = n
	}

	tokens := make(chan struct{}, limit)
	var group sync.WaitGroup
	group.Add(n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer group.Done()
			tokens <- struct{}{}
			defer func() { <-tokens }()
			fn(i)
		}(i)
	}

	group.Wait()
}

// EnsureAllCSS makes sure app.css is cached for every requested version.
//
// The run has two phases. First every version that needs a download is
// probed, and the versions with no GitHub release are reported and left out.
// Then the remaining versions are downloaded in ascending version order,
// several at a time.
//
// A version is left alone when a cached copy exists and is younger than the
// configured cache age, or when Force is set. A version with no GitHub
// release is recorded as unavailable and does not stop the run, so one
// missing release cannot hide the rest of the results. Those versions are
// probed again on the next run, so a release that appears later is picked up
// without any action.
//
// The returned result is never nil. Callers should check Failed to learn
// whether the cache is complete enough to trust.
func EnsureAllCSS(opts SweepOptions) *SweepResult {
	versions := SortVersions(opts.Versions)
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = DefaultSweepConcurrency
	}
	ttl := opts.TTL()

	result := &SweepResult{Requested: len(versions)}
	if len(versions) == 0 {
		return result
	}

	// Work out which versions still need a download.
	var pending, cached []string
	for _, v := range versions {
		if !opts.Force && CSSFresh(v, ttl) {
			cached = append(cached, v)
		} else {
			pending = append(pending, v)
		}
	}

	// Phase one: probe, so versions with no release never reach the download
	// stage. A probe that fails for another reason stays in the list, because
	// the download itself will report the real problem.
	available, skipped := probeReleases(pending, concurrency)
	for _, v := range skipped {
		result.Unavailable++
		result.UnavailableVersions = append(result.UnavailableVersions, v)
	}
	if opts.Log != nil && len(skipped) > 0 {
		_, _ = fmt.Fprintf(opts.Log, "Skipping %d versions with no GitHub release: %s\n",
			len(skipped), strings.Join(SortVersions(skipped), ", "))
	}

	// Phase two: download what is left, in ascending version order.
	downloaded, failed := downloadVersions(available, concurrency, opts)

	result.Cached = len(cached)
	result.Downloaded = len(downloaded)
	result.Failed = len(failed)
	result.DownloadedVersions = SortVersions(downloaded)
	result.FailedVersions = SortVersions(failed)

	if opts.Progress != nil {
		opts.Progress(result)
	}

	return result
}

// probeReleases asks GitHub which of the given versions have a release. It
// returns the versions to download and the versions to skip, both in
// ascending order.
func probeReleases(versions []string, concurrency int) (available, skipped []string) {
	if len(versions) == 0 {
		return nil, nil
	}

	skip := make([]bool, len(versions))
	ParallelFor(len(versions), concurrency, func(i int) {
		exists, err := ReleaseExists(versions[i])
		if err == nil && !exists {
			skip[i] = true
		}
	})

	for i, v := range versions {
		if skip[i] {
			skipped = append(skipped, v)
		} else {
			available = append(available, v)
		}
	}
	return available, skipped
}

// downloadVersions fetches the given versions, several at a time, and returns
// the ones that arrived and the ones that failed. Both lists come back in
// ascending version order.
func downloadVersions(versions []string, concurrency int, opts SweepOptions) (downloaded, failed []string) {
	if len(versions) == 0 {
		return nil, nil
	}

	var mu sync.Mutex
	done := 0

	ParallelFor(len(versions), concurrency, func(i int) {
		version := versions[i]
		status, err := ensureVersion(version, 0, true)

		mu.Lock()
		defer mu.Unlock()
		done++
		if opts.Log != nil {
			writeSweepLine(opts.Log, done, len(versions), version, status, err)
		}
		switch status {
		case SweepUnavailable:
		case SweepFailed:
			failed = append(failed, version)
		default:
			downloaded = append(downloaded, version)
		}
	})

	return SortVersions(downloaded), SortVersions(failed)
}

// ensureVersion caches a single version and reports what it did.
func ensureVersion(version string, ttl time.Duration, force bool) (SweepStatus, error) {
	if !force && CSSFresh(version, ttl) {
		return SweepCached, nil
	}
	_, err := ExtractCSSForce(version)
	switch {
	case err == nil:
		return SweepDownloaded, nil
	case errors.Is(err, ErrNoRelease):
		return SweepUnavailable, err
	default:
		return SweepFailed, err
	}
}

func writeSweepLine(w io.Writer, done, total int, version string, status SweepStatus, err error) {
	if err != nil && status != SweepUnavailable {
		_, _ = fmt.Fprintf(w, "[%d/%d] %s: %s: %v\n", done, total, version, status, err)
		return
	}
	_, _ = fmt.Fprintf(w, "[%d/%d] %s: %s\n", done, total, version, status)
}
