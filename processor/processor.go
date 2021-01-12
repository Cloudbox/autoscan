package processor

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/cloudbox/autoscan"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	Anchors    []string
	MinimumAge time.Duration
}

func New(c Config, db *sql.DB) (*Processor, error) {
	store, err := newDatastore(db)
	if err != nil {
		return nil, err
	}

	proc := &Processor{
		anchors:    c.Anchors,
		minimumAge: c.MinimumAge,
		store:      store,
	}
	return proc, nil
}

type Processor struct {
	anchors    []string
	minimumAge time.Duration
	store      *datastore
}

func (p *Processor) Add(scans ...autoscan.Scan) error {
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

func (p *Processor) callTargets(targets []autoscan.Target, scan autoscan.Scan) error {
	g := new(errgroup.Group)

	for _, target := range targets {
		target := target
		g.Go(func() error {
			return target.Scan(scan)
		})
	}

	return g.Wait()
}

func (p *Processor) Process(targets []autoscan.Target) error {
	scan, err := p.store.GetAvailableScan(p.minimumAge)
	if err != nil {
		return err
	}

	// Check whether all anchors are present
	for _, anchor := range p.anchors {
		if !fileExists(anchor) {
			return fmt.Errorf("%s: %w", anchor, autoscan.ErrAnchorUnavailable)
		}
	}

	// Fatal or Target Unavailable -> return original error
	err = p.callTargets(targets, scan)
	if err != nil {
		return err
	}

	err = p.store.Delete(scan)
	if err != nil {
		return err
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
