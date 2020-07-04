package processor

import (
	"database/sql"

	"github.com/cloudbox/autoscan"

	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

type datastore struct {
	db *sql.DB
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

	return &datastore{db: db}, nil
}

func (store *datastore) addScan(scan autoscan.Scan) error {
	_, err := store.db.Exec(sqlAddScan, scan.Path, scan.Priority, scan.Metadata.Provider, scan.Metadata.ID)
	return err
}

const sqlSchema = `
CREATE TABLE IF NOT EXISTS scan (
	"id" INTEGER NOT NULL PRIMARY KEY,
	"path" TEXT NOT NULL,
	"priority" INTEGER NOT NULL,
	"meta_provider" TEXT,
	"meta_id" TEXT
)
`

const sqlAddScan = `
INSERT INTO scan (path, priority, meta_provider, meta_id) VALUES (?, ?, NULLIF(?, ""), NULLIF(?, ""))
`
