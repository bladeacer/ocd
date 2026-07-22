package sources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bladeacer/obsi-css-diff/internal/models"
)

const electronRawURL = "https://raw.githubusercontent.com/Kilian/electron-to-chromium/master/full-versions.json"

type Electron struct {
	client *http.Client
}

func NewElectron() *Electron {
	return &Electron{client: &http.Client{Timeout: 15 * time.Second}}
}

func (e *Electron) Name() string { return "electron" }

func (e *Electron) Fetch() (models.ElectronMap, error) {
	resp, err := e.client.Get(electronRawURL)
	if err != nil {
		return nil, fmt.Errorf("electron fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("electron api status: %d", resp.StatusCode)
	}

	var raw map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("electron decode: %w", err)
	}

	return models.ElectronMap(raw), nil
}
