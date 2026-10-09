package core

import (
	"errors"
	"fmt"
	"io"
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
	// Progress, when set, is called once per version as it completes.
	// Callbacks run one at a time under an internal lock so that progress
	// lines stay in order, so they must not block or call back into this
	// package.
	Progress func(done, total int, version string, status SweepStatus, err error)
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

// EnsureAllCSS makes sure app.css is cached for every requested version.
//
// A version is left alone when a cached copy exists and is younger than the
// configured cache age, or when Force is set. Everything else is downloaded.
// A version with no GitHub release is recorded as unavailable and does not
// stop the run, so one missing release cannot hide the rest of the results.
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

	var (
		mu    sync.Mutex
		done  int
		group sync.WaitGroup
	)
	group.Add(len(versions))

	tokens := make(chan struct{}, concurrency)

	for _, v := range versions {
		go func(version string) {
			defer group.Done()
			tokens <- struct{}{}
			defer func() { <-tokens }()

			status, err := ensureVersion(version, ttl, opts.Force)

			mu.Lock()
			done++
			applySweepStatus(result, version, status)
			if opts.Log != nil {
				writeSweepLine(opts.Log, done, len(versions), version, status, err)
			}
			if opts.Progress != nil {
				opts.Progress(done, len(versions), version, status, err)
			}
			mu.Unlock()
		}(v)
	}

	group.Wait()

	sortResult(result)
	return result
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

// applySweepStatus records the outcome of one version. The caller holds the
// mutex.
func applySweepStatus(r *SweepResult, version string, status SweepStatus) {
	switch status {
	case SweepCached:
		r.Cached++
	case SweepDownloaded:
		r.Downloaded++
	case SweepUnavailable:
		r.Unavailable++
		r.UnavailableVersions = append(r.UnavailableVersions, version)
	case SweepFailed:
		r.Failed++
		r.FailedVersions = append(r.FailedVersions, version)
	}
}

// sortResult puts the version lists in ascending order. Downloads finish in
// an unpredictable order, so the lists are sorted for stable output.
func sortResult(r *SweepResult) {
	r.UnavailableVersions = SortVersions(r.UnavailableVersions)
	r.FailedVersions = SortVersions(r.FailedVersions)
}

func writeSweepLine(w io.Writer, done, total int, version string, status SweepStatus, err error) {
	if err != nil && status != SweepUnavailable {
		_, _ = fmt.Fprintf(w, "[%d/%d] %s: %s: %v\n", done, total, version, status, err)
		return
	}
	_, _ = fmt.Fprintf(w, "[%d/%d] %s: %s\n", done, total, version, status)
}
