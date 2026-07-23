package sources

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bladeacer/ocd/internal/models"
)

const testRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Obsidian Changelog</title>
<item>
<title>Obsidian 1.12.7 Desktop</title>
<description>Some changes</description>
<pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate>
</item>
<item>
<title>Obsidian 1.12.6 Desktop (Early access)</title>
<description>Early access build with Electron v28.1.0</description>
<pubDate>Mon, 15 Dec 2023 00:00:00 GMT</pubDate>
</item>
<item>
<title>Obsidian 1.12.5 Mobile</title>
<description>Mobile build</description>
<pubDate>Mon, 01 Dec 2023 00:00:00 GMT</pubDate>
</item>
</channel>
</rss>`

func TestRSSFetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(testRSS))
	}))
	defer ts.Close()

	orig := changelogURL
	changelogURL = ts.URL
	defer func() { changelogURL = orig }()

	r := NewRSS()
	versions, err := r.Fetch()
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}

	if versions[0].Version != "1.12.7" {
		t.Errorf("expected 1.12.7, got %s", versions[0].Version)
	}
	if versions[0].Type != models.Desktop {
		t.Errorf("expected Desktop, got %s", versions[0].Type)
	}
	if versions[0].IsEarly {
		t.Error("expected IsEarly=false for 1.12.7")
	}

	if versions[1].Version != "1.12.6" {
		t.Errorf("expected 1.12.6, got %s", versions[1].Version)
	}
	if !versions[1].IsEarly {
		t.Error("expected IsEarly=true for early access version")
	}
	if versions[1].Electron != "28.1.0" {
		t.Errorf("expected Electron 28.1.0, got %s", versions[1].Electron)
	}

	if versions[2].Version != "1.12.5" {
		t.Errorf("expected 1.12.5, got %s", versions[2].Version)
	}
	if versions[2].Type != models.Mobile {
		t.Errorf("expected Mobile, got %s", versions[2].Type)
	}
}

func TestRSSFetchHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	orig := changelogURL
	changelogURL = ts.URL
	defer func() { changelogURL = orig }()

	r := NewRSS()
	_, err := r.Fetch()
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestRSSFetchNoVersions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>No versions</title></channel></rss>`))
	}))
	defer ts.Close()

	orig := changelogURL
	changelogURL = ts.URL
	defer func() { changelogURL = orig }()

	r := NewRSS()
	_, err := r.Fetch()
	if err == nil {
		t.Fatal("expected error for no versions")
	}
}

func TestRSSFillElectron(t *testing.T) {
	versions := []models.RSSVersion{
		{Version: "1.0.0", Electron: ""},
		{Version: "1.0.1", Electron: "28.0.0"},
		{Version: "1.0.2", Electron: ""},
	}
	result := fillElectron(versions)
	if len(result) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(result))
	}
	if result[0].Electron != "28.0.0" {
		t.Errorf("expected 28.0.0 for 1.0.0, got %s", result[0].Electron)
	}
	if result[1].Electron != "28.0.0" {
		t.Errorf("expected 28.0.0 for 1.0.1, got %s", result[1].Electron)
	}
	if result[2].Electron != "13.0.0" {
		t.Errorf("expected 13.0.0 for 1.0.2, got %s", result[2].Electron)
	}
}

func TestRSSName(t *testing.T) {
	r := NewRSS()
	if r.Name() != "rss" {
		t.Errorf("expected rss, got %s", r.Name())
	}
}
