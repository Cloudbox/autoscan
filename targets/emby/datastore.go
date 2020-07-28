package emby

import (
	"database/sql"
	"fmt"

	// database driver
	_ "github.com/mattn/go-sqlite3"
)

func NewDatastore(path string) (*Datastore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %v", err)
	}

	return &Datastore{db: db}, nil
}

type Datastore struct {
	db *sql.DB
}

type Library struct {
	Name string
	Path string
}

func (d *Datastore) Libraries() ([]Library, error) {
	rows, err := d.db.Query(sqlSelectLibraries)
	if err != nil {
		return nil, fmt.Errorf("select libraries: %v", err)
	}

	defer rows.Close()

	libraries := make([]Library, 0)
	for rows.Next() {
		l := Library{}
		if err := rows.Scan(&l.Name, &l.Path); err != nil {
			return nil, fmt.Errorf("scan library row: %v", err)
		}

		libraries = append(libraries, l)
	}

	return libraries, nil
}

type MediaPart struct {
	ID          int
	File        string
	Size        uint64
}

func (d *Datastore) MediaPartByFile(path string) (*MediaPart, error) {
	mp := new(MediaPart)

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
