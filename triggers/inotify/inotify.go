package inotify

import (
	"fmt"
	"github.com/cloudbox/autoscan"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Priority  int                `yaml:"priority"`
	Verbosity string             `yaml:"verbosity"`
	Rewrite   []autoscan.Rewrite `yaml:"rewrite"`
	Include   []string           `yaml:"include"`
	Exclude   []string           `yaml:"exclude"`
	Paths     []struct {
		Path    string             `yaml:"path"`
		Rewrite []autoscan.Rewrite `yaml:"rewrite"`
		Include []string           `yaml:"include"`
		Exclude []string           `yaml:"exclude"`
	} `yaml:"paths"`
}

type daemon struct {
	callback autoscan.ProcessorFunc
	priority int
	paths    []path
	watcher  *fsnotify.Watcher
	log      zerolog.Logger
}

type path struct {
	Path     string
	Rewriter autoscan.Rewriter
	Allowed  autoscan.Filterer
	ScanTime func() time.Time
}

func New(c Config) (autoscan.Trigger, error) {
	l := autoscan.GetLogger(c.Verbosity).With().
		Str("trigger", "inotify").
		Logger()

	var paths []path
	for _, p := range c.Paths {
		p := p

		rewriter, err := autoscan.NewRewriter(append(p.Rewrite, c.Rewrite...))
		if err != nil {
			return nil, err
		}

		filterer, err := autoscan.NewFilterer(append(p.Include, c.Include...), append(p.Exclude, c.Exclude...))
		if err != nil {
			return nil, err
		}

		paths = append(paths, path{
			Path:     p.Path,
			Rewriter: rewriter,
			Allowed:  filterer,
		})
	}

	trigger := func(callback autoscan.ProcessorFunc) {
		d := daemon{
			log:      l,
			callback: callback,
			priority: c.Priority,
			paths:    paths,
		}

		// start job(s)
		if err := d.StartMonitoring(); err != nil {
			l.Error().
				Err(err).
				Msg("Failed initialising jobs")
			return
		}
	}

	return trigger, nil
}

func (d *daemon) StartMonitoring() error {
	// create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	d.watcher = watcher

	// setup watcher
	for _, p := range d.paths {
		if err := filepath.Walk(p.Path, d.walkFunc); err != nil {
			_ = d.watcher.Close()
			return err
		}
	}

	// start worker
	go d.worker()

	return nil
}

func (d *daemon) walkFunc(path string, fi os.FileInfo, err error) error {
	// ignore non-directory
	if !fi.Mode().IsDir() {
		return nil
	}

	if err := d.watcher.Add(path); err != nil {
		return fmt.Errorf("watch directory: %v: %w", path, err)
	}

	d.log.Trace().
		Str("path", path).
		Msg("Watching directory")

	return nil
}

func (d *daemon) getPathObject(path string) (*path, error) {
	for _, p := range d.paths {
		if strings.HasPrefix(path, p.Path) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("path object not found: %v", path)
}

func (d *daemon) worker() {
	// close watcher
	defer d.watcher.Close()

	// process events
	for {
		select {
		case event := <-d.watcher.Events:
			// new filesystem event
			d.log.Trace().
				Interface("event", event).
				Msg("Filesystem event")

			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				// create
				fi, err := os.Stat(event.Name)
				if err != nil {
					d.log.Error().
						Err(err).
						Str("path", event.Name).
						Msg("Failed retrieving filesystem info")
					continue
				}

				// watch new directories
				if fi.IsDir() {
					if err := filepath.Walk(event.Name, d.walkFunc); err != nil {
						d.log.Error().
							Err(err).
							Str("path", event.Name).
							Msg("Failed watching new directory")
					}

					continue
				}

			case event.Op&fsnotify.Rename == fsnotify.Rename, event.Op&fsnotify.Remove == fsnotify.Remove:
				// renamed / removed
			default:
				// ignore this event
				continue
			}

			// get path object
			p, err := d.getPathObject(event.Name)
			if err != nil {
				d.log.Error().
					Err(err).
					Str("path", event.Name).
					Msg("Failed determining path object")
				continue
			}

			// get file directory
			eventPath := event.Name
			if filepath.Ext(event.Name) != "" {
				// there was likely an extension, lets get the file path
				eventPath = filepath.Dir(event.Name)
			}

			// rewrite
			rewritten := p.Rewriter(eventPath)

			// filter
			if !p.Allowed(rewritten) {
				continue
			}

			// move to processor
			err = d.callback(autoscan.Scan{
				Folder:   filepath.Clean(rewritten),
				Priority: d.priority,
				Time:     time.Now(),
			})

			if err != nil {
				d.log.Error().
					Err(err).
					Str("path", rewritten).
					Msg("Failed moving scan to processor")
			} else {
				d.log.Info().
					Str("path", rewritten).
					Msg("Scan moved to processor")
			}

		case err := <-d.watcher.Errors:
			d.log.Error().
				Err(err).
				Msg("Failed receiving filesystem events")
		}
	}
}
