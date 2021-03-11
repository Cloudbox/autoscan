package processor

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"github.com/cloudbox/autoscan/migrate"
	"time"

	"github.com/cloudbox/autoscan"

	// sqlite3 driver
	_ "modernc.org/sqlite"
)

type datastore struct {
	*sql.DB
}

var (
	//go:embed migrations
	embedFS embed.FS
)

func newDatastore(db *sql.DB, mg *migrate.Migrator) (*datastore, error) {
	// migrations
	if err := mg.Migrate(&embedFS, "processor"); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &datastore{db}, nil
}

const sqlUpsert = `
INSERT INTO scan (folder, priority, time)
VALUES (?, ?, ?)
ON CONFLICT (folder) DO UPDATE SET
	priority = MAX(excluded.priority, scan.priority),
	time = excluded.time
`

func (store *datastore) upsert(tx *sql.Tx, scan autoscan.Scan) error {
	_, err := tx.Exec(sqlUpsert, scan.Folder, scan.Priority, scan.Time)
	return err
}

func (store *datastore) Upsert(scans []autoscan.Scan) error {
	tx, err := store.Begin()
	if err != nil {
		return err
	}

	for _, scan := range scans {
		if err = store.upsert(tx, scan); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				panic(rollbackErr)
			}

			return err
		}
	}

	return tx.Commit()
}

const sqlGetAvailableScan = `
SELECT folder, priority, time FROM scan
WHERE time < ?
ORDER BY priority DESC, time ASC
LIMIT 1
`

func (store *datastore) GetAvailableScan(minAge time.Duration) (autoscan.Scan, error) {
	row := store.QueryRow(sqlGetAvailableScan, now().Add(-1*minAge))

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

const sqlDelete = `
DELETE FROM scan WHERE folder=?
`

func (store *datastore) Delete(scan autoscan.Scan) error {
	_, err := store.Exec(sqlDelete, scan.Folder)
	if err != nil {
		return fmt.Errorf("delete: %s: %w", err, autoscan.ErrFatal)
	}

	return nil
}

var now = time.Now
