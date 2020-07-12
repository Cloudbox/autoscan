package processor

import (
	"database/sql"
	"time"

	"github.com/cloudbox/autoscan"

	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

type datastore struct {
	db  *sql.DB
	now func() time.Time
}

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
		db:  db,
		now: time.Now,
	}

	return store, nil
}

func (store *datastore) AddScan(scan autoscan.Scan) error {
	_, err := store.db.Exec(sqlAddScan, scan.Folder, scan.File, scan.Priority, scan.Size, scan.Metadata.Provider, scan.Metadata.ID, store.now())
	return err
}

const sqlSchema = `
CREATE TABLE IF NOT EXISTS scan (
	"folder" TEXT NOT NULL,
	"file" TEXT NOT NULL,
	"priority" INTEGER NOT NULL,
	"size" INTEGER NOT NULL,
	"time" DATETIME NOT NULL,
	"meta_provider" TEXT,
	"meta_id" TEXT,
	PRIMARY KEY(folder, file)
)
`

const sqlAddScan = `
INSERT INTO scan (folder, file, priority, size, meta_provider, meta_id, time)
VALUES (?, ?, ?, ?, NULLIF(?, ""), NULLIF(?, ""), ?)
ON CONFLICT (folder, file) DO UPDATE SET
	meta_provider = COALESCE(excluded.meta_provider, scan.meta_provider),
	meta_id = COALESCE(excluded.meta_id, scan.meta_id),
	priority = MAX(excluded.priority, scan.priority),
	size = excluded.size,
	time = excluded.time
`
