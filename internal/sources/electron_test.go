package sources

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestElectronFetch(t *testing.T) {
	data := map[string]string{
		"1.0.0": "1",
		"2.0.0": "2",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(data)
	}))
	defer ts.Close()

	orig := electronRawURL
	electronRawURL = ts.URL
	defer func() { electronRawURL = orig }()

	e := NewElectron()
	result, err := e.Fetch()
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}
	if result["1.0.0"] != "1" {
		t.Errorf("expected 1.0.0->1, got %s", result["1.0.0"])
	}
	if result["2.0.0"] != "2" {
		t.Errorf("expected 2.0.0->2, got %s", result["2.0.0"])
	}
}

func TestElectronFetchHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	orig := electronRawURL
	electronRawURL = ts.URL
	defer func() { electronRawURL = orig }()

	e := NewElectron()
	_, err := e.Fetch()
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestElectronFetchBadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer ts.Close()

	orig := electronRawURL
	electronRawURL = ts.URL
	defer func() { electronRawURL = orig }()

	e := NewElectron()
	_, err := e.Fetch()
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestElectronName(t *testing.T) {
	e := NewElectron()
	if e.Name() != "electron" {
		t.Errorf("expected electron, got %s", e.Name())
	}
}
