package emby

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/cloudbox/autoscan"

	// database driver
	_ "github.com/mattn/go-sqlite3"
)

func newDatastore(path string) (*datastore, error) {
	q := url.Values{}
	q.Set("mode", "ro")

	db, err := sql.Open("sqlite3", autoscan.DSN(path, q))
	if err != nil {
		return nil, fmt.Errorf("could not open database: %v", err)
	}

	return &datastore{db: db}, nil
}

type datastore struct {
	db *sql.DB
}

type library struct {
	Name string
	Path string
}

func (d *datastore) Libraries() ([]library, error) {
	rows, err := d.db.Query(sqlSelectLibraries)
	if err != nil {
		return nil, fmt.Errorf("select libraries: %v", err)
	}

	defer rows.Close()

	libraries := make([]library, 0)
	for rows.Next() {
		l := library{}
		if err := rows.Scan(&l.Name, &l.Path); err != nil {
			return nil, fmt.Errorf("scan library row: %v", err)
		}

		libraries = append(libraries, l)
	}

	return libraries, nil
}

type mediaPart struct {
	ID   int
	File string
	Size uint64
}

func (d *datastore) MediaPartByFile(path string) (*mediaPart, error) {
	mp := new(mediaPart)

	row := d.db.QueryRow(sqlSelectMediaPart, path)
	err := row.Scan(&mp.ID, &mp.File, &mp.Size)
	return mp, err
}

const (
	sqlSelectLibraries = `
SELECT
    mi.Name,
    mi.Path
FROM
    MediaItems mi
WHERE
    mi.type = 3 AND mi.ParentId = 1
`
	sqlSelectMediaPart = `
SELECT
    mi.Id,
    mi.Path,
    mi.Size
FROM
    MediaItems mi
WHERE
    mi.Path = $1
LIMIT
    1
`
)
