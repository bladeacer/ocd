package sources

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/models"
)

func TestFetchAllCacheHitsNoNetwork(t *testing.T) {
	orig := cache.CacheDir
	cache.CacheDir = t.TempDir()
	defer func() { cache.CacheDir = orig }()

	c, err := cache.New(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Set("rss_versions", []models.RSSVersion{{Version: "1.0.0", Type: models.Desktop}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("docker_versions", []models.DockerTag{{Version: "1.0.0"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("electron_versions", models.ElectronMap{"1.0.0": "90.0.0"}); err != nil {
		t.Fatal(err)
	}

	f := NewFetcher(c)

	origRSS := changelogURL
	origDocker := dockerHubAPI
	origElectron := electronRawURL
	changelogURL = "http://127.0.0.1:1/x"
	dockerHubAPI = "http://127.0.0.1:1/x"
	electronRawURL = "http://127.0.0.1:1/x"
	defer func() {
		changelogURL = origRSS
		dockerHubAPI = origDocker
		electronRawURL = origElectron
	}()

	result := f.FetchAll(false)
	if result.Error != nil {
		t.Fatalf("FetchAll should not error when all caches hit, got %v", result.Error)
	}
	if len(result.RSS) != 1 || result.RSS[0].Version != "1.0.0" {
		t.Errorf("expected cached RSS, got %v", result.RSS)
	}
	if len(result.Docker) != 1 || result.Docker[0].Version != "1.0.0" {
		t.Errorf("expected cached docker, got %v", result.Docker)
	}
	if result.Electron["1.0.0"] != "90.0.0" {
		t.Errorf("expected cached electron, got %v", result.Electron)
	}
}

func TestFetchAllForceBypassesCache(t *testing.T) {
	orig := cache.CacheDir
	cache.CacheDir = t.TempDir()
	defer func() { cache.CacheDir = orig }()

	c, err := cache.New(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Set("rss_versions", []models.RSSVersion{{Version: "9.9.9"}}); err != nil {
		t.Fatal(err)
	}

	f := NewFetcher(c)

	origRSS := changelogURL
	origDocker := dockerHubAPI
	origElectron := electronRawURL
	changelogURL = "http://127.0.0.1:1/x"
	dockerHubAPI = "http://127.0.0.1:1/x"
	electronRawURL = "http://127.0.0.1:1/x"
	defer func() {
		changelogURL = origRSS
		dockerHubAPI = origDocker
		electronRawURL = origElectron
	}()

	result := f.FetchAll(true)
	if result.Error == nil {
		t.Error("expected FetchAll to error when forced network fetch fails")
	}
}

func TestFetchAllPartialFailure(t *testing.T) {
	orig := cache.CacheDir
	cache.CacheDir = t.TempDir()
	defer func() { cache.CacheDir = orig }()

	c, err := cache.New(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Set("rss_versions", []models.RSSVersion{{Version: "1.0.0", Type: models.Desktop}}); err != nil {
		t.Fatal(err)
	}

	f := NewFetcher(c)

	origDocker := dockerHubAPI
	origElectron := electronRawURL
	dockerHubAPI = "http://127.0.0.1:1/x"
	electronRawURL = "http://127.0.0.1:1/x"
	defer func() {
		dockerHubAPI = origDocker
		electronRawURL = origElectron
	}()

	f.docker = &DockerHub{client: &http.Client{Timeout: 10 * time.Millisecond}}
	f.electron = &Electron{client: &http.Client{Timeout: 10 * time.Millisecond}}

	result := f.FetchAll(false)
	if result.Error == nil {
		t.Error("expected FetchAll to error when docker/electron network fails")
	}
	if len(result.RSS) != 1 {
		t.Errorf("expected RSS from cache, got %v", result.RSS)
	}
}

func TestFetchAllPaginatedDocker(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			resp := dockerTagResponse{
				Results: []dockerTagResult{
					{Name: "version-1.6.0", LastUpdated: "2024-02-01"},
				},
			}
			writeJSON(w, resp)
			return
		}
		resp := dockerTagResponse{
			Next:  serverURL + "?page=2",
			Results: []dockerTagResult{
				{Name: "version-1.5.0", LastUpdated: "2024-01-01"},
			},
		}
		writeJSON(w, resp)
	}))
	defer server.Close()
	serverURL = server.URL

	d := &DockerHub{client: server.Client()}
	orig := dockerHubAPI
	dockerHubAPI = server.URL
	defer func() { dockerHubAPI = orig }()

	tags, err := d.Fetch()
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	versions := []string{}
	for _, t := range tags {
		versions = append(versions, t.Version)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 tags across pages, got %d: %v", len(versions), versions)
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
