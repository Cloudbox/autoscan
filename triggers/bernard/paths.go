package bernard

import (
	"fmt"
	"github.com/m-rots/bernard"
	"github.com/m-rots/bernard/datastore"
	"github.com/m-rots/bernard/datastore/sqlite"
	"path/filepath"
)

type Paths struct {
	AddedFiles   []string
	ChangedFiles []string
	RemovedFiles []string
}

func NewPathsHook(driveID string, store *bds, diff *sqlite.Difference) (bernard.Hook, *Paths) {
	var paths Paths

	hook := func(drive datastore.Drive, files []datastore.File, folders []datastore.Folder, removed []string) error {
		folderMaps := getDiffFolderMaps(diff)
		fileMaps := getDiffFileMaps(diff)

		// get added file paths
		for _, f := range diff.AddedFiles {
			p, err := getFolderPath(store, driveID, f.Parent, folderMaps.Current)
			if err != nil {
				return fmt.Errorf("building file path for added file %v: %w", f.ID, err)
			}

			paths.AddedFiles = append(paths.AddedFiles, filepath.Join(p, f.Name))
		}

		// get changed file paths
		for _, f := range diff.ChangedFiles {
			// new path
			p, err := getFolderPath(store, driveID, f.New.Parent, folderMaps.Current)
			if err != nil {
				return fmt.Errorf("building file path for changed file %v: %w", f.New.ID, err)
			}

			paths.ChangedFiles = append(paths.ChangedFiles, filepath.Join(p, f.New.Name))

			// old (removed) path
			if !f.Old.Trashed && f.Old.ID != "" {
				p, err := getFolderPath(store, driveID, f.Old.Parent, folderMaps.Old)
				if err != nil {
					return fmt.Errorf("building removed file path for changed file %v: %w", f.Old.ID, err)
				}

				paths.RemovedFiles = append(paths.RemovedFiles, filepath.Join(p, f.Old.Name))
			}
		}

		// get removed file paths
		for _, f := range diff.RemovedFiles {
			p, err := getFolderPath(store, driveID, f.Parent, folderMaps.Old)
			if err != nil {
				return fmt.Errorf("building file path for removed file %v: %w", f.ID, err)
			}

			paths.RemovedFiles = append(paths.RemovedFiles, filepath.Join(p, f.Name))
		}

		// get new and old roots for changed folders
		newRoots, oldRoots := getRootChangedFolders(diff)

		// get changed file paths (descendants of newRoots)
		changedNewFiles, err := getChangedFolderFiles(store, driveID, newRoots, folderMaps.Current, fileMaps.Current)
		if err != nil {
			return fmt.Errorf("building changed folder descendant files: %w", err)
		}

		for _, f := range changedNewFiles {
			p, err := getFolderPath(store, driveID, f.Parent, folderMaps.Current)
			if err != nil {
				return fmt.Errorf("building changed file path for change folder descendant file %v: %w",
					f.ID, err)
			}

			paths.ChangedFiles = append(paths.ChangedFiles, filepath.Join(p, f.Name))
		}

		// get descendents of changed folders (old paths - removed)
		removedOldFiles, err := getChangedFolderFiles(store, driveID, oldRoots, folderMaps.Old, fileMaps.Old)
		if err != nil {
			return fmt.Errorf("building removed folder descendant files: %w", err)
		}

		for _, f := range removedOldFiles {
			if f.Trashed {
				continue
			}

			p, err := getFolderPath(store, driveID, f.Parent, folderMaps.Old)
			if err != nil {
				return fmt.Errorf("building removed file path for change folder descendant file %v: %w",
					f.ID, err)
			}

			paths.RemovedFiles = append(paths.RemovedFiles, filepath.Join(p, f.Name))
		}

		return nil
	}

	return hook, &paths
}

func getChangedFolderFiles(store *bds, driveID string, rootFolders []datastore.Folder,
	folderMap map[string]datastore.Folder, fileMap map[string]datastore.File) ([]datastore.File, error) {
	changedFiles := make([]datastore.File, 0)

	for _, folder := range rootFolders {
		// get descendants
		descendants, err := store.GetFolderDescendants(driveID, folder.ID)
		if err != nil {
			return nil, err
		}

		// iterate folder descendants (populating folderMap with missing)
		for foID, fo := range descendants.Folders {
			if _, ok := folderMap[foID]; ok {
				continue
			}

			folderMap[foID] = fo
		}

		// iterate descendants
		for fileID, file := range descendants.Files {
			// is there already a change for this file?
			if _, ok := fileMap[fileID]; ok {
				continue
			}

			fileMap[fileID] = file
			changedFiles = append(changedFiles, file)
		}
	}

	return changedFiles, nil
}

func getRootChangedFolders(diff *sqlite.Difference) ([]datastore.Folder, []datastore.Folder) {
	newFolders := make([]datastore.Folder, 0)
	oldFolders := make([]datastore.Folder, 0)

	for _, f := range diff.ChangedFolders {
		newFolders = append(newFolders, f.New)
		oldFolders = append(oldFolders, f.Old)
	}

	newRoots, _ := datastore.RootFolders(newFolders)
	oldRoots, _ := datastore.RootFolders(oldFolders)

	return newRoots, oldRoots
}

type diffFileMaps struct {
	Current map[string]datastore.File
	Old     map[string]datastore.File
}

func getDiffFileMaps(diff *sqlite.Difference) *diffFileMaps {
	currentFiles := make(map[string]datastore.File)
	oldFiles := make(map[string]datastore.File)

	for i, f := range diff.AddedFiles {
		currentFiles[f.ID] = diff.AddedFiles[i]
	}

	for i, f := range diff.ChangedFiles {
		currentFiles[f.New.ID] = diff.ChangedFiles[i].New
		oldFiles[f.Old.ID] = diff.ChangedFiles[i].Old
	}

	return &diffFileMaps{
		Current: currentFiles,
		Old:     oldFiles,
	}
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

	return &diffFolderMaps{
		Current: currentFolders,
		Old:     oldFolders,
	}
}

func getFolderPath(store *bds, driveId string, folderId string, folderMap map[string]datastore.Folder) (string, error) {
	path := ""

	// folderId == driveId
	if folderId == driveId {
		return path, nil
	}

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
