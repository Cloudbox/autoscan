package bernard

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/m-rots/bernard/datastore"
	"github.com/m-rots/bernard/datastore/sqlite"
)

type bds struct {
	*sqlite.Datastore
}

const sqlSelectFile = `SELECT id, name, parent, size, md5, trashed FROM file WHERE drive = $1 AND id = $2 LIMIT 1`

func (d *bds) GetFile(driveID string, fileID string) (*datastore.File, error) {
	f := new(datastore.File)

	row := d.DB.QueryRow(sqlSelectFile, driveID, fileID)
	err := row.Scan(&f.ID, &f.Name, &f.Parent, &f.Size, &f.MD5, &f.Trashed)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("%v: file not found: %w", fileID, sql.ErrNoRows)
	case err != nil:
		return nil, err
	default:
		break
	}

	return f, nil
}

const sqlSelectFolder = `SELECT id, name, trashed, parent FROM folder WHERE drive = $1 AND id = $2 LIMIT 1`

func (d *bds) GetFolder(driveID string, folderID string) (*datastore.Folder, error) {
	f := new(datastore.Folder)

	row := d.DB.QueryRow(sqlSelectFolder, driveID, folderID)
	err := row.Scan(&f.ID, &f.Name, &f.Trashed, &f.Parent)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("%v: folder not found: %w", folderID, sql.ErrNoRows)
	case err != nil:
		return nil, err
	default:
		break
	}

	return f, nil
}

const sqlSelectFolderDescendants = `
with cte_Folders as (
	-- Root Folder
	SELECT
	'folder' as [type]
	, f.id
	, f.drive
	, f.name
	, f.trashed
	, f.parent
	FROM folder f
	WHERE f.drive = $1 AND f.id = $2
	-- Descendant folders
	UNION
	SELECT
	'folder' as [type] 
	, f.id
	, f.drive
	, f.name
	, f.trashed
	, f.parent
	FROM cte_Folders cte
	JOIN folder f ON f.drive = cte.drive AND f.parent = cte.id
	WHERE cte.[type] = 'folder'
), cte_Combined as (
	-- Folders
	SELECT 
	*
	FROM cte_Folders cte
	
	-- Files
	UNION
	SELECT
	'file' as [type]
	, f.id
	, f.drive 
	, f.name 
	, f.trashed
	, f.parent 
	FROM cte_Folders cte
	JOIN file f ON f.drive = cte.drive AND f.parent = cte.id
	WHERE cte.[type] = 'folder'
)
SELECT DISTINCT
*
FROM cte_Combined cte
`

type folderDescendants struct {
	Folders map[string]datastore.Folder
	Files   map[string]datastore.File
}

func (d *bds) GetFolderDescendants(driveID string, folderID string) (*folderDescendants, error) {
	descendants := &folderDescendants{
		Folders: make(map[string]datastore.Folder),
		Files:   make(map[string]datastore.File),
	}

	if driveID == folderID {
		// never return descendants when folder is a drive
		return descendants, nil
	}

	rows, err := d.DB.Query(sqlSelectFolderDescendants, driveID, folderID)
	if errors.Is(err, sql.ErrNoRows) {
		return descendants, nil
	}

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	type Row struct {
		Type    string
		ID      string
		Drive   string
		Name    string
		Trashed bool
		Parent  string
	}

	for rows.Next() {
		desc := &Row{}

		err = rows.Scan(&desc.Type, &desc.ID, &desc.Drive, &desc.Name, &desc.Trashed, &desc.Parent)
		if err != nil {
			return nil, err
		}

		switch desc.Type {
		case "folder":
			descendants.Folders[desc.ID] = datastore.Folder{
				ID:      desc.ID,
				Name:    desc.Name,
				Parent:  desc.Parent,
				Trashed: desc.Trashed,
			}
		case "file":
			descendants.Files[desc.ID] = datastore.File{
				ID:      desc.ID,
				Name:    desc.Name,
				Parent:  desc.Parent,
				Trashed: desc.Trashed,
			}
		}
	}

	return descendants, rows.Err()
}
