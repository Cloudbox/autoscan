package bernard

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/l3uddz/bernard/datastore"
	"github.com/l3uddz/bernard/datastore/sqlite"
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

const sqlSelectDrive = `SELECT d.id, f.name, d.pageToken FROM drive d JOIN folder f ON f.id = d.id WHERE d.id = $1 LIMIT 1`

func (d *bds) GetDrive(driveID string) (*datastore.Drive, error) {
	drv := new(datastore.Drive)

	row := d.DB.QueryRow(sqlSelectDrive, driveID)
	err := row.Scan(&drv.ID, &drv.Name, &drv.PageToken)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("%v: drive not found: %w", driveID, sql.ErrNoRows)
	case err != nil:
		return nil, err
	default:
		break
	}

	return drv, nil
}
