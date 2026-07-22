package sources

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/models"
)

const fetchTimeout = 15 * time.Second

type Fetcher struct {
	rss      *RSS
	docker   *DockerHub
	electron *Electron
	cache    *cache.Store
}

func NewFetcher(c *cache.Store) *Fetcher {
	return &Fetcher{
		rss:      NewRSS(),
		docker:   NewDockerHub(),
		electron: NewElectron(),
		cache:    c,
	}
}

func (f *Fetcher) FetchAll(force bool) *models.FetchResult {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	var result models.FetchResult
	var g errgroup.Group

	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !force {
			var cached []models.RSSVersion
			if err := f.cache.Get("rss_versions", &cached); err == nil {
				result.RSS = cached
				return nil
			}
		}
		data, err := f.rss.Fetch()
		if err != nil {
			return err
		}
		if err := f.cache.Set("rss_versions", data); err != nil {
			return err
		}
		result.RSS = data
		return nil
	})

	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !force {
			var cached []models.DockerTag
			if err := f.cache.Get("docker_versions", &cached); err == nil {
				result.Docker = cached
				return nil
			}
		}
		data, err := f.docker.Fetch()
		if err != nil {
			return err
		}
		if err := f.cache.Set("docker_versions", data); err != nil {
			return err
		}
		result.Docker = data
		return nil
	})

	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if !force {
			var cached models.ElectronMap
			if err := f.cache.Get("electron_versions", &cached); err == nil {
				result.Electron = cached
				return nil
			}
		}
		data, err := f.electron.Fetch()
		if err != nil {
			return err
		}
		if err := f.cache.Set("electron_versions", data); err != nil {
			return err
		}
		result.Electron = data
		return nil
	})

	if err := g.Wait(); err != nil {
		result.Error = err
	}
	return &result
}
