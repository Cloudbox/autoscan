package processor

import (
	"database/sql"
	"errors"
	"time"

	"github.com/cloudbox/autoscan"

	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

type datastore struct {
	db *sql.DB
}

const sqlSchema = `
CREATE TABLE IF NOT EXISTS scan (
	"folder" TEXT NOT NULL,
	"file" TEXT NOT NULL,
	"priority" INTEGER NOT NULL,
	"time" DATETIME NOT NULL,
	"retries" INTEGER NOT NULL,
	"removed" BOOLEAN NOT NULL,
	PRIMARY KEY(folder, file)
)
`

func newDatastore(path string) (*datastore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(sqlSchema)
	if err != nil {
		return nil, err
	}

	store := &datastore{
		db: db,
	}

	return store, nil
}

const sqlUpsert = `
INSERT INTO scan (folder, file, priority, time, retries, removed)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (folder, file) DO UPDATE SET
	priority = MAX(excluded.priority, scan.priority),
	time = excluded.time,
	retries = excluded.retries,
	removed = min(excluded.removed, scan.removed)
`

func (store datastore) upsert(tx *sql.Tx, scan autoscan.Scan) error {
	_, err := tx.Exec(sqlUpsert, scan.Folder, scan.File, scan.Priority, scan.Time, scan.Retries, scan.Removed)
	return err
}

func (store datastore) Upsert(scans []autoscan.Scan) error {
	tx, err := store.db.Begin()
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

const sqlGetMatching = `
SELECT folder, file, priority, retries, removed FROM scan
WHERE folder = (
	SELECT folder
	FROM scan
	GROUP BY folder
	HAVING MAX(time) < ?
	ORDER BY priority DESC, time ASC
	LIMIT 1
)
`

func (store datastore) GetMatching(minAge time.Duration) (scans []autoscan.Scan, err error) {
	rows, err := store.db.Query(sqlGetMatching, now().Add(-1*minAge))
	if errors.Is(err, sql.ErrNoRows) {
		return scans, nil
	}

	if err != nil {
		return scans, err
	}

	defer rows.Close()
	for rows.Next() {
		scan := autoscan.Scan{}
		err = rows.Scan(&scan.Folder, &scan.File, &scan.Priority, &scan.Retries, &scan.Removed)
		if err != nil {
			return scans, err
		}

		scans = append(scans, scan)
	}

	return scans, rows.Err()
}

const sqlIncrementRetries = `
UPDATE scan
SET retries = retries + 1, time = ?
WHERE folder = ?
`

// Increment the retry count of all the children of a folder.
// Furthermore, we also update the timestamp to the current time
// so the children will not get scanned for 5 minutes.
func (store datastore) incrementRetries(tx *sql.Tx, folder string) error {
	_, err := tx.Exec(sqlIncrementRetries, now(), folder)
	return err
}

const sqlDeleteRetries = `
DELETE FROM scan
WHERE folder = ? AND retries > ?
`

func (store datastore) deleteRetries(tx *sql.Tx, folder string, maxRetries int) error {
	_, err := tx.Exec(sqlDeleteRetries, folder, maxRetries)
	return err
}

func (store datastore) Retry(folder string, maxRetries int) error {
	tx, err := store.db.Begin()
	if err != nil {
		return err
	}

	err = store.incrementRetries(tx, folder)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			panic(rbErr)
		}

		return err
	}

	err = store.deleteRetries(tx, folder, maxRetries)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			panic(rbErr)
		}

		return err
	}

	return tx.Commit()
}

const sqlGetAll = `
SELECT folder, file, priority, retries, removed FROM scan
`

func (store datastore) GetAll() (scans []autoscan.Scan, err error) {
	rows, err := store.db.Query(sqlGetAll)
	if errors.Is(err, sql.ErrNoRows) {
		return scans, nil
	}

	if err != nil {
		return scans, err
	}

	defer rows.Close()
	for rows.Next() {
		scan := autoscan.Scan{}
		err = rows.Scan(&scan.Folder, &scan.File, &scan.Priority, &scan.Retries, &scan.Removed)
		if err != nil {
			return scans, err
		}

		scans = append(scans, scan)
	}

	return scans, rows.Err()
}

const sqlDelete = `
DELETE FROM scan
WHERE folder=? AND file=?
`

func (store datastore) delete(tx *sql.Tx, scan autoscan.Scan) error {
	_, err := tx.Exec(sqlDelete, scan.Folder, scan.File)
	return err
}

func (store datastore) Delete(scans []autoscan.Scan) error {
	tx, err := store.db.Begin()
	if err != nil {
		return err
	}

	for _, scan := range scans {
		if err = store.delete(tx, scan); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				panic(rollbackErr)
			}

			return err
		}
	}

	return tx.Commit()
}

// todo: remove once tests have been refactored for support of Time on Scan struct
var now = time.Now
