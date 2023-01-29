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
	DbSign string
}

var (
	//go:embed migrations
	migrations embed.FS
)

func newDatastore(db *sql.DB, dbSign string, mg *migrate.Migrator) (*datastore, error) {
	// migrations
	if err := mg.Migrate(&migrations, "processor"); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &datastore{db, dbSign}, nil
}

func sqlUpsert(dbSign string) (string) {
	return fmt.Sprintf(`
		INSERT INTO scan (folder, priority, time)
		VALUES (%s, %s, %s)
		ON CONFLICT (folder) DO UPDATE SET
			priority = MAX(excluded.priority, scan.priority),
			time = excluded.time
		`, dbSign)
}

func (store *datastore) upsert(tx *sql.Tx, dbSign string, scan autoscan.Scan) error {
	_, err := tx.Exec(sqlUpsert(dbSign), scan.Folder, scan.Priority, scan.Time)
	return err
}

func (store *datastore) Upsert(scans []autoscan.Scan) error {
	tx, err := store.Begin()
	if err != nil {
		return err
	}

	for _, scan := range scans {
		if err = store.upsert(tx, store.DbSign, scan); err != nil {
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

func sqlGetAvailableScan(dbSign string) (string) {
	return fmt.Sprintf(`
		SELECT folder, priority, time FROM scan
		WHERE time < %s
		ORDER BY priority DESC, time ASC
		LIMIT 1
		`, dbSign)
}

func (store *datastore) GetAvailableScan(minAge time.Duration) (autoscan.Scan, error) {
	row := store.QueryRow(sqlGetAvailableScan(store.DbSign), now().Add(-1*minAge))

	scan := autoscan.Scan{}
	err := row.Scan(&scan.Folder, &scan.Priority, &scan.Time)
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

func sqlDelete(dbSign string) (string) {
	return fmt.Sprintf(`
		DELETE FROM scan WHERE folder=%s
		`, dbSign)
}

func (store *datastore) Delete(scan autoscan.Scan) error {
	_, err := store.Exec(sqlDelete(store.DbSign), scan.Folder)
	if err != nil {
		return fmt.Errorf("delete: %s: %w", err, autoscan.ErrFatal)
	}

	return nil
}

var now = time.Now
