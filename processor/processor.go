package processor

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"github.com/cloudbox/autoscan"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	Anchors       []string
	DatastorePath string
	MaxRetries    int
	MinimumAge    time.Duration
	Include       []string
	Exclude       []string
}

func New(c Config) (*Processor, error) {
	store, err := newDatastore(c.DatastorePath)
	if err != nil {
		return nil, err
	}

	includes := make([]regexp.Regexp, 0)
	excludes := make([]regexp.Regexp, 0)

	for _, pattern := range c.Include {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed compiling include: %v: %w", pattern, err)
		}
		includes = append(includes, *re)
	}

	for _, pattern := range c.Exclude {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed compiling exclude: %v: %w", pattern, err)
		}
		excludes = append(excludes, *re)
	}

	proc := &Processor{
		anchors:    c.Anchors,
		maxRetries: c.MaxRetries,
		minimumAge: c.MinimumAge,
		store:      store,
		includes:   includes,
		excludes:   excludes,
	}
	return proc, nil
}

type Processor struct {
	anchors    []string
	maxRetries int
	minimumAge time.Duration
	store      *datastore
	includes   []regexp.Regexp
	excludes   []regexp.Regexp
}

func (p *Processor) matchesFilter(scan autoscan.Scan, filters []regexp.Regexp) bool {
	scanPath := filepath.Join(scan.Folder, scan.File)
	for _, re := range filters {
		if re.MatchString(scanPath) {
			return true
		}
	}

	return false
}

func (p *Processor) Add(scans ...autoscan.Scan) error {
	switch {
	case len(p.includes) > 0:
		// includes only
		filteredScans := make([]autoscan.Scan, 0)
		for _, scan := range scans {
			if p.matchesFilter(scan, p.includes) {
				filteredScans = append(filteredScans, scan)
				continue
			}
		}

		return p.store.Upsert(filteredScans)

	case len(p.excludes) > 0:
		// excludes only
		filteredScans := make([]autoscan.Scan, 0)
		for _, scan := range scans {
			if p.matchesFilter(scan, p.excludes) {
				continue
			}
			filteredScans = append(filteredScans, scan)
		}

		return p.store.Upsert(filteredScans)
	}

	// no filtering of scans
	return p.store.Upsert(scans)
}

// CheckAvailability checks whether all targets are available.
// If one target is not available, the error will return.
func (p *Processor) CheckAvailability(targets []autoscan.Target) error {
	g := new(errgroup.Group)

	for _, target := range targets {
		target := target
		g.Go(func() error {
			return target.Available()
		})
	}

	return g.Wait()
}

func (p *Processor) callTargets(targets []autoscan.Target, scans []autoscan.Scan) error {
	g := new(errgroup.Group)

	for _, target := range targets {
		target := target
		g.Go(func() error {
			return target.Scan(scans)
		})
	}

	return g.Wait()
}

func (p *Processor) Process(targets []autoscan.Target) error {
	// Get children of the same folder with the highest priority and oldest date.
	scans, err := p.store.GetMatching(p.minimumAge)
	if err != nil {
		return fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	// When no scans are currently available,
	// return the ErrNoScans.
	if len(scans) == 0 {
		return fmt.Errorf("%w", autoscan.ErrNoScans)
	}

	// Check whether all anchors are present
	for _, anchor := range p.anchors {
		if !fileExists(anchor) {
			return fmt.Errorf("%s: %w", anchor, autoscan.ErrAnchorUnavailable)
		}
	}

	// Check which files exist on the file system.
	// We do not want to try to scan non-existing files.
	var existingScans []autoscan.Scan
	for _, scan := range scans {
		if scan.Removed != fileExists(path.Join(scan.Folder, scan.File)) {
			existingScans = append(existingScans, scan)
		}
	}

	// When no files currently exist on the file system,
	// then we want to exit early and retry later.
	if len(existingScans) == 0 {
		if err = p.store.Retry(scans[0].Folder, p.maxRetries); err != nil {
			return fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
		}

		return nil
	}

	// 1. do stuff with existingScans
	// Fatal or Target Unavailable -> return original error
	err = p.callTargets(targets, existingScans)
	if err != nil {
		return err
	}

	// 2. remove existingScans from datastore
	if err = p.store.Delete(existingScans); err != nil {
		return fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	// 3. update non-existing scans with retry +1
	if err = p.store.Retry(scans[0].Folder, p.maxRetries); err != nil {
		return fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	return nil
}

var fileExists = func(fileName string) bool {
	info, err := os.Stat(fileName)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
