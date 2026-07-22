package sources

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mmcdole/gofeed"

	"github.com/bladeacer/ocd/internal/models"
)

const changelogURL = "https://obsidian.md/changelog.xml"

type RSS struct{}

func NewRSS() *RSS {
	return &RSS{}
}

func (r *RSS) Name() string { return "rss" }

func (r *RSS) Fetch() ([]models.RSSVersion, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(changelogURL)
	if err != nil {
		return nil, fmt.Errorf("rss fetch: %w", err)
	}

	versionRe := regexp.MustCompile(`(\d+\.\d+\.\d+)`)
	electronRe := regexp.MustCompile(`(?i)electron v?(\d+\.\d+\.\d+)`)

	versions := make([]models.RSSVersion, 0, len(feed.Items))

	for _, entry := range feed.Items {
		vMatch := versionRe.FindStringSubmatch(entry.Title)
		if vMatch == nil {
			continue
		}
		vNum := vMatch[1]

		vType := models.Mobile
		if strings.Contains(entry.Title, "Desktop") {
			vType = models.Desktop
		}

		isEarly := strings.Contains(entry.Title, "(Early access)") ||
			strings.Contains(entry.Title, "(Insider)")

		var electronVer string
		if entry.Content != "" {
			elMatch := electronRe.FindStringSubmatch(entry.Content)
			if elMatch != nil {
				electronVer = elMatch[1]
			}
		} else if entry.Description != "" {
			elMatch := electronRe.FindStringSubmatch(entry.Description)
			if elMatch != nil {
				electronVer = elMatch[1]
			}
		}

		date := entry.Updated
		if date == "" {
			date = entry.Published
		}

		versions = append(versions, models.RSSVersion{
			Version:  vNum,
			Type:     vType,
			Date:     date,
			Electron: electronVer,
			Title:    entry.Title,
			IsEarly:  isEarly,
		})
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found in RSS feed")
	}

	versions = fillElectron(versions)
	return versions, nil
}

func fillElectron(versions []models.RSSVersion) []models.RSSVersion {
	sorted := make([]models.RSSVersion, len(versions))
	copy(sorted, versions)

	lastElectron := "13.0.0"
	for i := len(sorted) - 1; i >= 0; i-- {
		if sorted[i].Electron != "" {
			lastElectron = sorted[i].Electron
		} else {
			sorted[i].Electron = lastElectron
		}
	}
	return sorted
}
