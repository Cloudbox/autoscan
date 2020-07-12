package processor

import (
	"github.com/cloudbox/autoscan"
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

func (p *Processor) AddScan(scan autoscan.Scan) error {
	return p.store.AddScan(scan)
}

func (p *Processor) ProcessTargets(targets []autoscan.Target) {
	// read from database

	// process one by one
	// trigger the targets
}
