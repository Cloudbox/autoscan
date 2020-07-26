package plex

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

type LibraryType int

const (
	Movie LibraryType = 1
	TV    LibraryType = 2
	Music LibraryType = 8
)

type Library struct {
	ID   int
	Type LibraryType
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
		if err := rows.Scan(&l.ID, &l.Type, &l.Name, &l.Path); err != nil {
			return nil, fmt.Errorf("scan library row: %v", err)
		}

		libraries = append(libraries, l)
	}

	return libraries, nil
}

type MediaPart struct {
	ID          int
	DirectoryID int
	File        string
	Size        uint64
}

func (d *Datastore) MediaPartByFile(path string) (*MediaPart, error) {
	mp := new(MediaPart)

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
`
)
