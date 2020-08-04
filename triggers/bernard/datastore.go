package bernard

import (
	"database/sql"
	"errors"
	"github.com/m-rots/bernard/datastore"
	"github.com/m-rots/bernard/datastore/sqlite"
)

type bds struct {
	*sqlite.Datastore
}

func (d *bds) GetFile(driveID string, fileID string) (*datastore.File, error) {
	f := new(datastore.File)

	row := d.DB.QueryRow(sqlSelectFile, driveID, fileID)
	err := row.Scan(&f.ID, &f.Name, &f.Parent, &f.Size, &f.MD5, &f.Trashed)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrFileNotFound
	case err != nil:
		return nil, err
	default:
		break
	}

	return f, nil
}

func (d *bds) GetFolder(driveID string, fileID string) (*datastore.Folder, error) {
	f := new(datastore.Folder)

	row := d.DB.QueryRow(sqlSelectFolder, driveID, fileID)
	err := row.Scan(&f.ID, &f.Name, &f.Trashed, &f.Parent)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrFolderNotFound
	case err != nil:
		return nil, err
	default:
		break
	}

	return f, nil
}

var (
	/* Google */
	ErrFileNotFound   = errors.New("file not found")
	ErrFolderNotFound = errors.New("folder not found")
	ErrDriveNotFound  = errors.New("drive not found")
)

const (
	/* Google */
	// - selects
	sqlSelectFile   = `SELECT id, name, parent, size, md5, trashed FROM file WHERE drive = $1 AND id = $2 LIMIT 1`
	sqlSelectFolder = `SELECT id, name, trashed, parent FROM folder WHERE drive = $1 AND id = $2 LIMIT 1`
	//sqlSelectDrive          = `SELECT * FROM drive WHERE id = ? LIMIT 1`
	//sqlSelectDriveWithName  = `SELECT d.*, f.name FROM drive d JOIN folder f ON f.drive = d.id AND f.id = d.id WHERE d.id = ? LIMIT 1`
	//sqlSelectDriveTotalSize = `SELECT SUM(f.size) as total_size FROM file f WHERE f.drive = ? AND f.trashed = 0`
)
