package plex

import (
	"database/sql"
	"errors"
	"fmt"

	// database driver
	_ "github.com/mattn/go-sqlite3"
)

func New(path string) (*Datastore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", ErrDatabase)
	}

	return &Datastore{db: db}, nil
}

type Datastore struct {
	db *sql.DB
}

var ErrDatabase = errors.New("datastore: database related error")

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
		return nil, fmt.Errorf("select libraries: %w", ErrDatabase)
	}

	defer rows.Close()

	libraries := make([]Library, 0)
	for rows.Next() {
		l := Library{}
		if err := rows.Scan(&l.ID, &l.Type, &l.Name, &l.Path); err != nil {
			return nil, fmt.Errorf("scan library row: %w", ErrDatabase)
		}

		libraries = append(libraries, l)
	}

	return libraries, nil
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
)
