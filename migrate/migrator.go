package migrate

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/oriser/regroup"
	"modernc.org/sqlite"
	"github.com/lib/pq"
)

type Migrator struct {
	db     *sql.DB
	dbType string
	dir    string

	re *regroup.ReGroup
}

/* Credits to https://github.com/Boostport/migration */

func New(db *sql.DB, dbType string, dir string) (*Migrator, error) {
	var err error

	m := &Migrator{
		db:     db,
		dbType: dbType,
		dir:    dir,
	}

	// validate supported driver
	if dbType == "postgres" {
		if _, ok := db.Driver().(*pq.Driver); !ok {
			return nil, errors.New("database instance is not using the postgres driver")
		}
	} else {
		if _, ok := db.Driver().(*sqlite.Driver); !ok {
			return nil, errors.New("database instance is not using the sqlite driver")
		}
	}

	// verify schema
	if err = m.verify(); err != nil {
		return nil, fmt.Errorf("verify: %w", err)
	}

	// compile migration regexp
	m.re, err = regroup.Compile(`(?P<Version>\d+)\w?(?P<Name>.+)?\.sql`)
	if err != nil {
		return nil, fmt.Errorf("regexp: %w", err)
	}

	return m, nil
}

func (m *Migrator) Migrate(fs *embed.FS, component string) error {
	// parse migrations
	migrations, err := m.parse(fs)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if len(migrations) == 0 {
		return nil
	}

	// get current migration versions
	versions, err := m.versions(component, m.dbType)
	if err != nil {
		return fmt.Errorf("versions: %v: %w", component, err)
	}

	// migrate
	for _, mg := range migrations {
		// already have this version?
		if _, exists := versions[mg.Version]; exists {
			continue
		}

		// migrate
		if err := m.exec(component, m.dbType, mg); err != nil {
			return fmt.Errorf("migrate: %v: %w", mg.Filename, err)
		}
	}

	return nil
}
