package migrate

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"github.com/oriser/regroup"
	"modernc.org/sqlite"
)

type Migrator struct {
	db  *sql.DB
	dir string

	re *regroup.ReGroup
}

/* Credits to https://github.com/Boostport/migration */

func New(db *sql.DB, dir string) (*Migrator, error) {
	var err error

	m := &Migrator{
		db:  db,
		dir: dir,
	}

	// validate supported driver
	if _, ok := db.Driver().(*sqlite.Driver); !ok {
		return nil, errors.New("database instance is not using the sqlite driver")
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
	versions, err := m.versions(component)
	if err != nil {
		return fmt.Errorf("versions: %v: %w", component, err)
	}

	// migrate
	for _, migration := range migrations {
		// already have this version?
		if _, exists := versions[migration.Version]; exists {
			continue
		}

		// migrate
		if err := m.exec(component, migration); err != nil {
			return fmt.Errorf("migrate: %v: %w", migration.Filename, err)
		}
	}

	return nil
}
