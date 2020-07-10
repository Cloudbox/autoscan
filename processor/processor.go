package processor

import (
	"fmt"

	"github.com/cloudbox/autoscan"
)

func New(dbPath string) (*Processor, error) {
	store, err := newDatastore(dbPath)
	if err != nil {
		return nil, err
	}

	return &Processor{store: store}, nil
}

type Processor struct {
	store *datastore
}

func (p *Processor) ProcessTriggers(scans chan autoscan.Scan) {
	for {
		scan, ok := <-scans
		if !ok {
			break
		}
		fmt.Printf("%+v\n", scan)

		// write to database
		if err := p.store.AddScan(scan); err != nil {
			panic(err)
		}
	}
}

func (p *Processor) ProcessTargets(targets []autoscan.Target) {
	// read from database

	// process one by one
	// trigger the targets
}
