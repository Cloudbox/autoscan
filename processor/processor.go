package processor

import (
	"os"
	"path"
	"time"

	"github.com/cloudbox/autoscan"
	"golang.org/x/sync/errgroup"
)

func New(dbPath string) (*Processor, error) {
	store, err := newDatastore(dbPath)
	if err != nil {
		return nil, err
	}

	proc := &Processor{
		store: store,
	}
	return proc, nil
}

type Processor struct {
	store *datastore
}

func (p *Processor) Add(scans ...autoscan.Scan) error {
	return p.store.Upsert(scans)
}

func callTargets(targets []autoscan.Target, scans []autoscan.Scan) error {
	g := new(errgroup.Group)

	for _, target := range targets {
		g.Go(func() error {
			return target.Scan(scans)
		})
	}

	return g.Wait()
}

func (p *Processor) Process(targets []autoscan.Target) error {
	// Get children of the same folder with the highest priority and oldest date.
	scans, err := p.store.GetMatching()
	if err != nil {
		return err
	}

	// TODO: remove items with more than 5 retries.

	// Only sleep when no scans are currently available in the datastore.
	if len(scans) == 0 {
		sleep(10 * time.Second)
		return nil
	}

	// Check which files exist on the file system.
	// We do not want to try to scan non-existing files.
	var existingScans []autoscan.Scan
	for _, scan := range scans {
		if fileExists(path.Join(scan.Folder, scan.File)) {
			existingScans = append(existingScans, scan)
		}
	}

	// When no files currently exist on the file system,
	// then we want to exit early and retry later.
	if len(existingScans) == 0 {
		if err = p.store.IncrementRetries(scans[0].Folder); err != nil {
			return err
		}

		return nil
	}

	// 1. do stuff with existingScans
	err = callTargets(targets, existingScans)
	if err != nil {
		// Something produced an error... >:(
		// Either the target is unavailable, or a scan broke the target. ¯\_(ツ)_/¯
		// Now we have to retry all of the files again at a later date!
		if incrementErr := p.store.IncrementRetries(scans[0].Folder); incrementErr != nil {
			return incrementErr
		}

		return err
	}

	// 2. remove existingScans from datastore
	if err = p.store.Delete(existingScans); err != nil {
		return err
	}

	// 3. update non-existing scans with retry +1
	if err = p.store.IncrementRetries(scans[0].Folder); err != nil {
		return err
	}

	return nil
}

var sleep = func(dur time.Duration) {
	time.Sleep(dur)
}

var fileExists = func(fileName string) bool {
	info, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}
