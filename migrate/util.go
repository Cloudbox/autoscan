package migrate

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
)

type migration struct {
	Version  int    `regroup:"Version"`
	Name     string `regroup:"Name"`
	Filename string
	Schema   string
}

func (m *Migrator) verify() error {
	if _, err := m.db.Exec(sqlSchema); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	return nil
}

func (m *Migrator) versions(component string) (map[int]bool, error) {
	rows, err := m.db.Query(sqlVersions, component)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	versions := make(map[int]bool, 0)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		versions[version] = true
	}

	return versions, nil
}

func (m *Migrator) exec(component string, migration *migration) (err error) {
	// begin tx
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	// commit - rollback
	defer func(tx *sql.Tx) {
		// roll back
		if err != nil {
			if errRb := tx.Rollback(); errRb != nil {
				err = fmt.Errorf("rollback: %v: %w", errRb, err)
			}
			return
		}

		// commit
		if errCm := tx.Commit(); err != nil {
			err = fmt.Errorf("commit: %w", errCm)
		}
	}(tx)

	// exec migration
	if _, err := tx.Exec(migration.Schema); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	// insert migration version
	if _, err := tx.Exec(sqlInsertVersion, component, migration.Version); err != nil {
		return fmt.Errorf("schema_migration: %w", err)
	}

	return nil
}

func (m *Migrator) parse(fs *embed.FS) ([]*migration, error) {
	// parse migrations from filesystem
	files, err := fs.ReadDir(m.dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	// parse migrations
	migrations := make([]*migration, 0)
	for _, f := range files {
		// skip dirs
		if f.IsDir() {
			continue
		}

		// parse migration
		md := new(migration)
		if err := m.re.MatchToTarget(f.Name(), md); err != nil {
			return nil, fmt.Errorf("parse migration: %w", err)
		}

		b, err := fs.ReadFile(filepath.Join(m.dir, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("read migration: %w", err)
		}
		md.Schema = string(b)
		md.Filename = f.Name()

		// set migration
		migrations = append(migrations, md)
	}

	// sort migrations
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

const (
	sqlSchema        = `CREATE TABLE IF NOT EXISTS schema_migration (component VARCHAR(255) NOT NULL, version INTEGER NOT NULL, PRIMARY KEY (component, version))`
	sqlVersions      = `SELECT version FROM schema_migration WHERE component = ?`
	sqlInsertVersion = `INSERT INTO schema_migration (component, version) VALUES (?, ?)`
)
