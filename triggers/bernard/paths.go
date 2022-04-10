package bernard

import (
	"fmt"
	"path/filepath"

	"github.com/l3uddz/bernard"
	"github.com/l3uddz/bernard/datastore"
	"github.com/l3uddz/bernard/datastore/sqlite"
)

type Paths struct {
	NewFolders []string
	OldFolders []string
}

func NewPathsHook(driveID string, store *bds, diff *sqlite.Difference) (bernard.Hook, *Paths) {
	var paths Paths

	hook := func(drive datastore.Drive, files []datastore.File, folders []datastore.Folder, removed []string) error {
		// get folders from diff (that we are interested in)
		parents, err := getDiffFolders(store, driveID, diff)
		if err != nil {
			return fmt.Errorf("getting parents: %w", err)
		}

		// get roots from folders
		rootNewFolders, _ := datastore.RootFolders(parents.New)
		rootOldFolders, _ := datastore.RootFolders(parents.Old)

		// get new/changed paths
		for _, folder := range rootNewFolders {
			p, err := getFolderPath(store, driveID, folder.ID, parents.FolderMaps.Current)
			if err != nil {
				return fmt.Errorf("building folder path: %v: %w", folder.ID, err)
			}

			paths.NewFolders = append(paths.NewFolders, p)
		}

		// get removed paths
		for _, folder := range rootOldFolders {
			p, err := getFolderPath(store, driveID, folder.ID, parents.FolderMaps.Old)
			if err != nil {
				return fmt.Errorf("building old folder path: %v: %w", folder.ID, err)
			}

			paths.OldFolders = append(paths.OldFolders, p)
		}

		return nil
	}

	return hook, &paths
}

type diffFolderMaps struct {
	Current map[string]datastore.Folder
	Old     map[string]datastore.Folder
}

func getDiffFolderMaps(diff *sqlite.Difference) *diffFolderMaps {
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

	for i, f := range diff.RemovedFolders {
		oldFolders[f.ID] = diff.RemovedFolders[i]
	}

	return &diffFolderMaps{
		Current: currentFolders,
		Old:     oldFolders,
	}
}

type Parents struct {
	New        []datastore.Folder
	Old        []datastore.Folder
	FolderMaps *diffFolderMaps
}

func getDiffFolders(store *bds, driveId string, diff *sqlite.Difference) (*Parents, error) {
	folderMaps := getDiffFolderMaps(diff)

	newParents := make(map[string]datastore.Folder)
	oldParents := make(map[string]datastore.Folder)

	// changed folders
	for _, folder := range diff.ChangedFolders {
		newParents[folder.New.ID] = folder.New
		oldParents[folder.Old.ID] = folder.Old
	}

	// removed folders
	for _, folder := range diff.RemovedFolders {
		oldParents[folder.ID] = folder
	}

	// added files
	for _, file := range diff.AddedFiles {
		folder, err := getFolder(store, driveId, file.Parent, folderMaps.Current)
		if err != nil {
			return nil, fmt.Errorf("added file: %w", err)
		}

		newParents[folder.ID] = *folder
	}

	// changed files
	for _, file := range diff.ChangedFiles {
		// current
		currentFolder, err := getFolder(store, driveId, file.New.Parent, folderMaps.Current)
		if err != nil {
			return nil, fmt.Errorf("changed new file: %w", err)
		}

		newParents[currentFolder.ID] = *currentFolder

		// old
		oldFolder, err := getFolder(store, driveId, file.Old.Parent, folderMaps.Old)
		if err != nil {
			return nil, fmt.Errorf("changed old file: %w", err)
		}

		oldParents[oldFolder.ID] = *oldFolder
	}

	// removed files
	for _, file := range diff.RemovedFiles {
		oldFolder, err := getFolder(store, driveId, file.Parent, folderMaps.Old)
		if err != nil {
			return nil, fmt.Errorf("removed file: %w", err)
		}

		oldParents[oldFolder.ID] = *oldFolder
	}

	// create Parents object
	p := &Parents{
		New:        make([]datastore.Folder, 0),
		Old:        make([]datastore.Folder, 0),
		FolderMaps: folderMaps,
	}

	for _, folder := range newParents {
		p.New = append(p.New, folder)
	}

	for _, folder := range oldParents {
		p.Old = append(p.Old, folder)
	}

	return p, nil
}

func getFolder(store *bds, driveId string, folderId string, folderMap map[string]datastore.Folder) (*datastore.Folder, error) {
	// find folder in map
	if folder, ok := folderMap[folderId]; ok {
		return &folder, nil
	}

	if folderId == driveId {
		folder := datastore.Folder{
			ID:      driveId,
			Name:    "",
			Parent:  "",
			Trashed: false,
		}

		folderMap[driveId] = folder
		return &folder, nil
	}

	// search datastore
	folder, err := store.GetFolder(driveId, folderId)
	if err != nil {
		return nil, fmt.Errorf("could not get folder: %v: %w", folderId, err)
	}

	// add folder to map
	folderMap[folder.ID] = *folder

	return folder, nil
}

func getFolderPath(store *bds, driveId string, folderId string, folderMap map[string]datastore.Folder) (string, error) {
	path := ""

	// folderId == driveId
	if folderId == driveId {
		return "/", nil
	}

	// get top folder
	topFolder, ok := folderMap[folderId]
	if !ok {
		f, err := store.GetFolder(driveId, folderId)
		if err != nil {
			return filepath.Join("/", path), fmt.Errorf("could not get folder %v: %w", folderId, err)
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
				return filepath.Join("/", path), fmt.Errorf("could not get folder %v: %w", nextFolderId, err)
			}

			f = *df
			folderMap[f.ID] = f
		}

		path = filepath.Join(f.Name, path)
		nextFolderId = f.Parent
	}

	return filepath.Join("/", path), nil
}
