package bernard

import (
	"fmt"

	"github.com/l3uddz/bernard"
	"github.com/l3uddz/bernard/datastore"
	"github.com/l3uddz/bernard/datastore/sqlite"
)

func NewPostProcessBernardDiff(driveID string, store *bds, diff *sqlite.Difference) bernard.Hook {
	hook := func(drive datastore.Drive, files []datastore.File, folders []datastore.Folder, removed []string) error {
		// dont include removes for files already known as trashed
		for i := 0; i < len(diff.RemovedFiles); i++ {
			df := diff.RemovedFiles[i]

			ef, err := store.GetFile(driveID, df.ID)
			if err != nil {
				return fmt.Errorf("retrieving file (id: %v): %w", df.ID, err)
			}

			switch {
			case ef.Trashed && df.Trashed:
				// this removed file was already known as trashed (removed to us)
				diff.RemovedFiles = append(diff.RemovedFiles[:i], diff.RemovedFiles[i+1:]...)
				i--
			}
		}

		// dont include removes for folders already known as trashed
		for i := 0; i < len(diff.RemovedFolders); i++ {
			df := diff.RemovedFolders[i]

			ef, err := store.GetFolder(driveID, df.ID)
			if err != nil {
				return fmt.Errorf("retrieving folder (id: %v): %w", df.ID, err)
			}

			switch {
			case ef.Trashed && df.Trashed:
				// this removed folder was already known as trashed (removed to us)
				diff.RemovedFolders = append(diff.RemovedFolders[:i], diff.RemovedFolders[i+1:]...)
				i--
			}
		}

		// remove changed files that were trashed or un-trashed
		for i := 0; i < len(diff.ChangedFiles); i++ {
			df := diff.ChangedFiles[i].New
			ef := diff.ChangedFiles[i].Old

			switch {
			case ef.Trashed && !df.Trashed:
				// existing state was trashed, but new state is not
				diff.AddedFiles = append(diff.AddedFiles, df)
				diff.ChangedFiles = append(diff.ChangedFiles[:i], diff.ChangedFiles[i+1:]...)
				i--
			case !ef.Trashed && df.Trashed:
				// new state is trashed, existing state is not
				diff.RemovedFiles = append(diff.RemovedFiles, df)
				diff.ChangedFiles = append(diff.ChangedFiles[:i], diff.ChangedFiles[i+1:]...)
				i--
			}
		}

		for i := 0; i < len(diff.ChangedFolders); i++ {
			df := diff.ChangedFolders[i].New
			ef := diff.ChangedFolders[i].Old

			switch {
			case ef.Trashed && !df.Trashed:
				// existing state was trashed, but new state is not
				diff.AddedFolders = append(diff.AddedFolders, df)
				diff.ChangedFolders = append(diff.ChangedFolders[:i], diff.ChangedFolders[i+1:]...)
				i--
			case !ef.Trashed && df.Trashed:
				// new state is trashed, existing state is not
				diff.RemovedFolders = append(diff.RemovedFolders, df)
				diff.ChangedFolders = append(diff.ChangedFolders[:i], diff.ChangedFolders[i+1:]...)
				i--
			}
		}

		return nil
	}

	return hook
}
