package sources

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/models"
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

func TestFetchAllCached(t *testing.T) {
	c := cache.New(0)
	rssData := []models.RSSVersion{
		{Version: "1.12.7", Type: models.Desktop, Date: "2024-01-01", Electron: "28.0.0"},
	}
	if err := c.Set("rss_versions", rssData); err != nil {
		t.Fatalf("cache Set: %v", err)
	}
	dockerData := []models.DockerTag{
		{Version: "1.12.7", Tag: "version-1.12.7", LastUpdated: "2024-01-01"},
	}
	if err := c.Set("docker_versions", dockerData); err != nil {
		t.Fatalf("cache Set: %v", err)
	}
	electronData := models.ElectronMap{"28.0.0": "120"}
	if err := c.Set("electron_versions", electronData); err != nil {
		t.Fatalf("cache Set: %v", err)
	}

	f := NewFetcher(c)
	result := f.FetchAll(false)
	if result.Error != nil {
		t.Fatalf("FetchAll error: %v", result.Error)
	}
	if len(result.RSS) != 1 {
		t.Errorf("expected 1 RSS version, got %d", len(result.RSS))
	}
	if len(result.Docker) != 1 {
		t.Errorf("expected 1 Docker tag, got %d", len(result.Docker))
	}
	if result.Electron == nil {
		t.Error("expected Electron map")
	}
}

func TestFetchAllWithServers(t *testing.T) {
	rssSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>Test</title><item><title>Obsidian 1.0.0 Desktop</title><description>test</description><pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate></item></channel></rss>`))
	}))
	defer rssSrv.Close()

	dockerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := dockerTagResponse{
			Results: []dockerTagResult{
				{Name: "1.0.0", LastUpdated: "2024-01-01"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer dockerSrv.Close()

	electronSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"1.0.0": "1"})
	}))
	defer electronSrv.Close()

	origRSS := changelogURL
	origDocker := dockerHubAPI
	origElectron := electronRawURL
	changelogURL = rssSrv.URL
	dockerHubAPI = dockerSrv.URL
	electronRawURL = electronSrv.URL
	defer func() {
		changelogURL = origRSS
		dockerHubAPI = origDocker
		electronRawURL = origElectron
	}()

	c := cache.New(0)
	f := NewFetcher(c)
	result := f.FetchAll(true)
	if result.Error != nil {
		t.Fatalf("FetchAll error: %v", result.Error)
	}
	if len(result.RSS) != 1 {
		t.Errorf("expected 1 RSS version, got %d", len(result.RSS))
	}
}

func TestFetchAllCancel(t *testing.T) {
	c := cache.New(0)
	f := NewFetcher(c)
	result := f.FetchAll(false)
	if result.Error == nil {
		t.Log("FetchAll succeeded (all sources may be cached or reachable)")
	}
}
