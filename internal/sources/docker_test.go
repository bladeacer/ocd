package sources

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDockerHubParse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := dockerTagResponse{
			Results: []dockerTagResult{
				{Name: "version-1.5.3", LastUpdated: "2024-01-01"},
				{Name: "version-1.5.3-ls123", LastUpdated: "2024-01-02"},
				{Name: "latest", LastUpdated: "2024-01-03"},
				{Name: "1.4.0", LastUpdated: "2024-01-04"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	d := &DockerHub{client: server.Client()}
	originalURL := dockerHubAPI
	dockerHubAPI = server.URL
	defer func() { dockerHubAPI = originalURL }()

	tags, err := d.Fetch()
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %+v", len(tags), tags)
	}

	if tags[0].Version != "1.5.3" {
		t.Errorf("expected version 1.5.3, got %s", tags[0].Version)
	}
	if tags[1].Version != "1.4.0" {
		t.Errorf("expected version 1.4.0, got %s", tags[1].Version)
	}
}

func TestDockerHubEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := dockerTagResponse{}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	d := &DockerHub{client: server.Client()}
	originalURL := dockerHubAPI
	dockerHubAPI = server.URL
	defer func() { dockerHubAPI = originalURL }()

	tags, err := d.Fetch()
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

func TestDockerHubHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	d := &DockerHub{client: server.Client()}
	originalURL := dockerHubAPI
	dockerHubAPI = server.URL
	defer func() { dockerHubAPI = originalURL }()

	_, err := d.Fetch()
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}
