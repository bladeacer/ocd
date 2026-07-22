package sources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bladeacer/ocd/internal/models"
)

var dockerHubAPI = "https://hub.docker.com/v2/repositories/linuxserver/obsidian/tags"

type DockerHub struct {
	client *http.Client
}

func NewDockerHub() *DockerHub {
	return &DockerHub{client: &http.Client{Timeout: 15 * time.Second}}
}

func (d *DockerHub) Name() string { return "docker" }

type dockerTagResult struct {
	Name        string `json:"name"`
	LastUpdated string `json:"last_updated"`
}

type dockerTagResponse struct {
	Count   int               `json:"count"`
	Next    string            `json:"next"`
	Results []dockerTagResult `json:"results"`
}

var noisePattern = regexp.MustCompile(`(latest|develop|amd64|arm64|-ls)`)

func (d *DockerHub) Fetch() ([]models.DockerTag, error) {
	var tags []models.DockerTag
	url := dockerHubAPI

	for url != "" {
		resp, err := d.client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("docker hub request: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("docker hub api status: %d", resp.StatusCode)
		}

		var page dockerTagResponse
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("docker hub decode: %w", err)
		}
		resp.Body.Close()

		for _, result := range page.Results {
			if noisePattern.MatchString(result.Name) {
				continue
			}
			cleanVersion := strings.TrimPrefix(result.Name, "version-")
			cleanVersion = strings.Split(cleanVersion, "-")[0]

			vParts := regexp.MustCompile(`\d+`).FindAllString(cleanVersion, -1)
			if len(vParts) == 0 || vParts[0] == "0" {
				continue
			}

			tags = append(tags, models.DockerTag{
				Version:     cleanVersion,
				Tag:         result.Name,
				LastUpdated: result.LastUpdated,
			})
		}
		url = page.Next
	}

	return tags, nil
}
