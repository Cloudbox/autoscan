package bernard

import (
	"fmt"
	"github.com/m-rots/bernard"
	"github.com/m-rots/bernard/datastore"
	"github.com/m-rots/bernard/datastore/sqlite"
	"path/filepath"
)

type Paths struct {
	sqlite.Difference

	AddedFiles   []string
	ChangedFiles []string
	RemovedFiles []string

	//AddedFolders   []string
	//ChangedFolders []string
	//RemovedFolders []string
}

func NewPathsHook(driveID string, store *bds, diff *sqlite.Difference) (bernard.Hook, *Paths) {
	var paths Paths

	hook := func(drive datastore.Drive, files []datastore.File, folders []datastore.Folder, removed []string) error {
		folderMaps := getDiffFolderMaps(diff)

		// get added file paths
		for _, f := range diff.AddedFiles {
			p, err := getFolderPath(store, driveID, f.Parent, folderMaps.Current)
			if err != nil {
				return fmt.Errorf("failed building file path for added file %v: %w", f.ID, err)
			}

			paths.AddedFiles = append(paths.AddedFiles, filepath.Join(p, f.Name))
		}

		// get changed file paths
		for _, f := range diff.ChangedFiles {
			p, err := getFolderPath(store, driveID, f.New.Parent, folderMaps.Current)
			if err != nil {
				return fmt.Errorf("failed building file path for changed file %v: %w", f.New.ID, err)
			}

			paths.ChangedFiles = append(paths.ChangedFiles, filepath.Join(p, f.New.Name))
		}

		// get removed file paths
		for _, f := range diff.RemovedFiles {
			p, err := getFolderPath(store, driveID, f.Parent, folderMaps.Old)
			if err != nil {
				return fmt.Errorf("failed building file path for removed file %v: %w", f.ID, err)
			}

			paths.RemovedFiles = append(paths.RemovedFiles, filepath.Join(p, f.Name))
		}

		return nil
	}

	return hook, &paths
}

type FolderMaps struct {
	Current map[string]datastore.Folder
	Old     map[string]datastore.Folder
}

func getDiffFolderMaps(diff *sqlite.Difference) *FolderMaps {
	currentFolders := make(map[string]datastore.Folder)
	oldFolders := make(map[string]datastore.Folder)

	for i, f := range diff.AddedFolders {
		currentFolders[f.ID] = diff.AddedFolders[i]
		oldFolders[f.ID] = diff.AddedFolders[i]
	}

	for i, f := range diff.ChangedFolders {
		currentFolders[f.New.ID] = diff.ChangedFolders[i].New
		oldFolders[f.Old.ID] = diff.ChangedFolders[i].Old
	}

	return &FolderMaps{
		Current: currentFolders,
		Old:     oldFolders,
	}
}

func getFolderPath(store *bds, driveId string, folderId string, folderMap map[string]datastore.Folder) (string, error) {
	path := ""

	// get top folder
	topFolder, ok := folderMap[folderId]
	if !ok {
		f, err := store.GetFolder(driveId, folderId)
		if err != nil {
			return path, fmt.Errorf("could not get folder %v: %w", folderId, err)
		}

		topFolder = *f
	}

	// set logic variables
	path = topFolder.Name
	nextFolderId := topFolder.Parent

	// get folder paths
	for nextFolderId != "" && nextFolderId != driveId {
		f, ok := folderMap[nextFolderId]
		if !ok {
			df, err := store.GetFolder(driveId, nextFolderId)
			if err != nil {
				return path, fmt.Errorf("could not get folder %v: %w", nextFolderId, err)
			}

			f = *df
			folderMap[f.ID] = f
		}

		path = filepath.Join(f.Name, path)
		nextFolderId = f.Parent
	}

	return path, nil
}
