package plex

import (
	"database/sql"
	"fmt"

	// database driver
	_ "github.com/mattn/go-sqlite3"
)

func NewDatastore(path string) (*datastore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %v", err)
	}

	return &datastore{db: db}, nil
}

type datastore struct {
	db *sql.DB
}

type libraryType int

const (
	libraryMovie libraryType = 1
	libraryTV    libraryType = 2
	libraryMusic libraryType = 8
)

type library struct {
	ID   int
	Type libraryType
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
		if err := rows.Scan(&l.ID, &l.Type, &l.Name, &l.Path); err != nil {
			return nil, fmt.Errorf("scan library row: %v", err)
		}

		libraries = append(libraries, l)
	}

	return libraries, nil
}

type mediaPart struct {
	ID          int
	DirectoryID int
	File        string
	Size        uint64
}

func (d *datastore) MediaPartByFile(path string) (*mediaPart, error) {
	mp := new(mediaPart)

	row := d.db.QueryRow(sqlSelectMediaPart, path)
	err := row.Scan(&mp.ID, &mp.DirectoryID, &mp.File, &mp.Size)
	return mp, err
}

const (
	sqlSelectLibraries = `
SELECT
    ls.id,
    ls.section_type,
    ls.name,
    sl.root_path
FROM
    library_sections ls
    JOIN section_locations sl ON sl.library_section_id = ls.id
`
	sqlSelectMediaPart = `
SELECT
    mp.id,
    mp.directory_id,
    mp.file,
    mp.size
FROM
    media_parts mp
WHERE
    mp.file = $1
LIMIT 1
`
)
