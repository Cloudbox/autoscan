package inotify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"

	"github.com/cloudbox/autoscan"
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
	paths    []path
	watcher  *fsnotify.Watcher
	queue    *queue
	log      zerolog.Logger
}

type path struct {
	Path     string
	Rewriter autoscan.Rewriter
	Allowed  autoscan.Filterer
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
			paths:    paths,
			queue:    newQueue(callback, l, c.Priority),
		}

		// start job(s)
		if err := d.startMonitoring(); err != nil {
			l.Error().
				Err(err).
				Msg("Failed initialising jobs")
			return
		}
	}

	return trigger, nil
}

func (d *daemon) startMonitoring() error {
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
	// handle error
	if err != nil {
		return fmt.Errorf("walk func: %v: %w", path, err)
	}

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

			// rewrite
			rewritten := p.Rewriter(event.Name)

			// filter
			if !p.Allowed(rewritten) {
				continue
			}

			// get directory where path has an extension
			if filepath.Ext(rewritten) != "" {
				// there was most likely a file extension, use the directory
				rewritten = filepath.Dir(rewritten)
			}

			// move to queue
			d.queue.inputs <- rewritten

		case err := <-d.watcher.Errors:
			d.log.Error().
				Err(err).
				Msg("Failed receiving filesystem events")
		}
	}
}

type queue struct {
	callback autoscan.ProcessorFunc
	log      zerolog.Logger
	priority int
	inputs   chan string
	scans    map[string]time.Time
	lock     *sync.Mutex
}

func newQueue(cb autoscan.ProcessorFunc, log zerolog.Logger, priority int) *queue {
	q := &queue{
		callback: cb,
		log:      log,
		priority: priority,
		inputs:   make(chan string),
		scans:    make(map[string]time.Time),
		lock:     &sync.Mutex{},
	}

	go q.worker()

	return q
}

func (q *queue) add(path string) {
	// acquire lock
	q.lock.Lock()
	defer q.lock.Unlock()

	// queue scan task
	q.scans[path] = time.Now().Add(10 * time.Second)
}

func (q *queue) worker() {
	for {
		select {
		case path, ok := <-q.inputs:
			if !ok {
				// channel closed
				return
			}

			// add path to queue
			q.add(path)

		default:
			// process queue
			q.process()
		}
	}
}

func (q *queue) process() {
	// acquire lock
	q.lock.Lock()
	defer q.lock.Unlock()

	// sleep if no scans queued
	if len(q.scans) == 0 {
		time.Sleep(100 * time.Millisecond)
		return
	}

	// move scans to processor
	for p, t := range q.scans {
		// time has not elapsed
		if time.Now().Before(t) {
			continue
		}

		// move to processor
		err := q.callback(autoscan.Scan{
			Folder:   filepath.Clean(p),
			Priority: q.priority,
			Time:     time.Now(),
		})

		if err != nil {
			q.log.Error().
				Err(err).
				Str("path", p).
				Msg("Failed moving scan to processor")
		} else {
			q.log.Info().
				Str("path", p).
				Msg("Scan moved to processor")
		}

		// remove queued scan
		delete(q.scans, p)
	}
}
