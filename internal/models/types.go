package models

import "time"

type VersionType string

const (
	Desktop VersionType = "Desktop"
	Mobile  VersionType = "Mobile"
)

type RSSVersion struct {
	Version   string      `json:"version"`
	Type      VersionType `json:"type"`
	Date      string      `json:"date"`
	Electron  string      `json:"electron"`
	Title     string      `json:"title"`
	IsEarly   bool        `json:"is_early"`
	Chromium  string      `json:"chromium,omitempty"`
	DockerTag string      `json:"docker_tag,omitempty"`
}

type DockerTag struct {
	Version     string `json:"version"`
	Tag         string `json:"tag"`
	LastUpdated string `json:"last_updated"`
}

type ElectronMap map[string]string

type CacheEntry struct {
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

type FetchResult struct {
	RSS      []RSSVersion
	Docker   []DockerTag
	Electron ElectronMap
	Error    error
}

type ExtractResult struct {
	Version string
	Path    string
	Error   error
}

type DiffResult struct {
	VersionA string
	VersionB string
	Diff     string
	HasDiff  bool
	Error    error
}
