package processor

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/aleksasiriski/autoscan"
	"github.com/aleksasiriski/autoscan/migrate"

	// sqlite3 driver
	_ "modernc.org/sqlite"
	// postgresql driver
	_ "github.com/lib/pq"
)

type datastore struct {
	*sql.DB
	dbType string
}

//go:embed migrations/sqlite
var migrationsSqlite embed.FS

//go:embed migrations/postgres
var migrationsPostgres embed.FS

func newDatastore(db *sql.DB, dbType string, mg *migrate.Migrator) (*datastore, error) {
	switch dbType {
	case "sqlite":
		{
			// migrations/sqlite
			if err := mg.Migrate(&migrationsSqlite, "processor"); err != nil {
				return nil, fmt.Errorf("migrate: %w", err)
			}
		}
	case "postgres":
		{
			// migrations/postgres
			if err := mg.Migrate(&migrationsPostgres, "processor"); err != nil {
				return nil, fmt.Errorf("migrate: %w", err)
			}
		}
	default:
		return &datastore{}, fmt.Errorf("incorrect database type")
	}
	return &datastore{db, dbType}, nil
}

func sqlSelectVersion(dbType string) (string, error) {
	switch dbType {
	case "sqlite":
		return "", nil
	case "postgres":
		return `SELECT version()`, nil
	default:
		return "", fmt.Errorf("incorrect database type")
	}
}

func (store *datastore) SelectVersion() (string, error) {
	sqlSelectVersion, err := sqlSelectVersion(store.dbType)
	if err != nil {
		return "", err
	}

	version := ""
	if store.dbType != "sqlite" {
		row := store.QueryRow(sqlSelectVersion)
		err = row.Scan(&version)
		if err != nil {
			return version, fmt.Errorf("select version: %w", err)
		}
	}

	return version, nil
}

func sqlUpsert(dbType string) (string, error) {
	switch dbType {
	case "sqlite":
		return `
				INSERT INTO scan (folder, priority, time)
				VALUES (?, ?, ?)
				ON CONFLICT (folder) DO UPDATE SET
					priority = MAX(excluded.priority, scan.priority),
					time = excluded.time
				`, nil
	case "postgres":
		return `
				INSERT INTO scan (folder, priority, time)
				VALUES ($1, $2, $3)
				ON CONFLICT (folder) DO UPDATE SET
					priority = GREATEST(excluded.priority, scan.priority),
					time = excluded.time
				`, nil
	default:
		return "", fmt.Errorf("incorrect database type")
	}
}

func (store *datastore) upsert(tx *sql.Tx, dbType string, scan autoscan.Scan) error {
	sqlUpsert, err := sqlUpsert(dbType)
	if err != nil {
		return err
	}

	_, err = tx.Exec(sqlUpsert, scan.Folder, scan.Priority, scan.Time)
	return err
}

func (store *datastore) Upsert(scans []autoscan.Scan) error {
	tx, err := store.Begin()
	if err != nil {
		return err
	}

	for _, scan := range scans {
		if err = store.upsert(tx, store.dbType, scan); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				panic(rollbackErr)
			}

			return err
		}
	}

	return tx.Commit()
}

const sqlGetScansRemaining = `SELECT COUNT(folder) FROM scan`

func (store *datastore) GetScansRemaining() (int, error) {
	row := store.QueryRow(sqlGetScansRemaining)

	remaining := 0
	err := row.Scan(&remaining)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return remaining, nil
	case err != nil:
		return remaining, fmt.Errorf("get remaining scans: %v: %w", err, autoscan.ErrFatal)
	}

	return remaining, nil
}

func sqlGetAvailableScan(dbType string) (string, error) {
	switch dbType {
	case "sqlite":
		return `
				SELECT folder, priority, time FROM scan
				WHERE time < ?
				ORDER BY priority DESC, time ASC
				LIMIT 1
				`, nil
	case "postgres":
		return `
				SELECT folder, priority, time FROM scan
				WHERE time < $1
				ORDER BY priority DESC, time ASC
				LIMIT 1
				`, nil
	default:
		return "", fmt.Errorf("incorrect database type")
	}
}

func (store *datastore) GetAvailableScan(minAge time.Duration) (autoscan.Scan, error) {
	scan := autoscan.Scan{}

	sqlGetAvailableScan, err := sqlGetAvailableScan(store.dbType)
	if err != nil {
		return scan, err
	}

	row := store.QueryRow(sqlGetAvailableScan, now().Add(-1*minAge))
	err = row.Scan(&scan.Folder, &scan.Priority, &scan.Time)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return scan, autoscan.ErrNoScans
	case err != nil:
		return scan, fmt.Errorf("get matching: %s: %w", err, autoscan.ErrFatal)
	}

	return scan, nil
}

const sqlGetAll = `
SELECT folder, priority, time FROM scan
`

func (store *datastore) GetAll() (scans []autoscan.Scan, err error) {
	rows, err := store.Query(sqlGetAll)
	if err != nil {
		return scans, err
	}

	defer rows.Close()
	for rows.Next() {
		scan := autoscan.Scan{}
		err = rows.Scan(&scan.Folder, &scan.Priority, &scan.Time)
		if err != nil {
			return scans, err
		}

		scans = append(scans, scan)
	}

	return scans, rows.Err()
}

func sqlDelete(dbType string) (string, error) {
	switch dbType {
	case "sqlite":
		return `DELETE FROM scan WHERE folder=?`, nil
	case "postgres":
		return `DELETE FROM scan WHERE folder=$1`, nil
	default:
		return "", fmt.Errorf("incorrect database type")
	}
}

func (store *datastore) Delete(scan autoscan.Scan) error {
	sqlDelete, err := sqlDelete(store.dbType)
	if err != nil {
		return fmt.Errorf("delete: %s: %w", err, autoscan.ErrFatal)
	}

	_, err = store.Exec(sqlDelete, scan.Folder)
	if err != nil {
		return fmt.Errorf("delete: %s: %w", err, autoscan.ErrFatal)
	}

	return nil
}

var now = time.Now
