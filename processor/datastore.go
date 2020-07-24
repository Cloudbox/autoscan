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
	"size" INTEGER NOT NULL,
	"time" DATETIME NOT NULL,
	"retries" INTEGER NOT NULL,
	"meta_provider" TEXT,
	"meta_id" TEXT,
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
INSERT INTO scan (folder, file, priority, size, meta_provider, meta_id, time, retries)
VALUES (?, ?, ?, ?, NULLIF(?, ""), NULLIF(?, ""), ?, ?)
ON CONFLICT (folder, file) DO UPDATE SET
	meta_provider = COALESCE(excluded.meta_provider, scan.meta_provider),
	meta_id = COALESCE(excluded.meta_id, scan.meta_id),
	priority = MAX(excluded.priority, scan.priority),
	size = excluded.size,
	time = excluded.time,
	retries = excluded.retries
`

func (store datastore) upsert(tx *sql.Tx, scan autoscan.Scan) error {
	_, err := tx.Exec(sqlUpsert, scan.Folder, scan.File, scan.Priority, scan.Size, scan.Metadata.Provider, scan.Metadata.ID, now(), scan.Retries)
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
SELECT folder, file, priority, size, retries, IFNULL(meta_provider, ""), IFNULL(meta_id, "") FROM scan
WHERE folder = (
	SELECT folder
	FROM scan
	GROUP BY folder
	HAVING MAX(time) < ?
	ORDER BY priority DESC, time ASC
	LIMIT 1
)
`

func (store datastore) GetMatching() (scans []autoscan.Scan, err error) {
	rows, err := store.db.Query(sqlGetMatching, now().Add(-5*time.Minute))
	if errors.Is(err, sql.ErrNoRows) {
		return scans, nil
	}

	if err != nil {
		return scans, err
	}

	defer rows.Close()
	for rows.Next() {
		scan := autoscan.Scan{}
		err = rows.Scan(&scan.Folder, &scan.File, &scan.Priority, &scan.Size, &scan.Retries, &scan.Metadata.Provider, &scan.Metadata.ID)
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
func (store datastore) IncrementRetries(folder string) error {
	_, err := store.db.Exec(sqlIncrementRetries, now(), folder)
	return err
}

const sqlGetAll = `
SELECT folder, file, priority, size, retries, IFNULL(meta_provider, ""), IFNULL(meta_id, "") FROM scan
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
		err = rows.Scan(&scan.Folder, &scan.File, &scan.Priority, &scan.Size, &scan.Retries, &scan.Metadata.Provider, &scan.Metadata.ID)
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

var now = time.Now
