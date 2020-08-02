package emby

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudbox/autoscan"
)

func (t target) Scan(scans []autoscan.Scan) error {
	// ensure scan tasks present (should never fail)
	if len(scans) == 0 {
		return nil
	}

	// determine library for this scan
	scanFolder := t.rewrite(scans[0].Folder)

	lib, err := t.getScanLibrary(scanFolder)
	if err != nil {
		t.log.Warn().
			Err(err).
			Msg("No target library found")
		return fmt.Errorf("%v: %w", err, autoscan.ErrRetryScan)
	}

	// check for at-least one missing/changed file
	process := false
	for _, s := range scans {
		targetFilePath := t.rewrite(filepath.Join(s.Folder, s.File))

		targetFile, err := t.api.MediaPartByFile(lib.ID, targetFilePath)
		if err != nil {
			if errors.Is(err, autoscan.ErrNotFoundInTarget) {
				// local file not found in target
				t.log.Debug().
					Str("path", targetFilePath).
					Str("library", lib.Name).
					Msg("At least one local file did not exist in the target")

				process = true
				break
			}

			// handle expected errors
			if errors.Is(err, autoscan.ErrTargetUnavailable) || errors.Is(err, autoscan.ErrRetryScan) {
				return fmt.Errorf("could not check emby: %w", err)
			}

			// unexpected error
			return fmt.Errorf("could not check emby: %v: %w", err, autoscan.ErrFatal)
		}

		// local file was found in target
		if targetFile.Size != s.Size {
			// local file did not match in target
			t.log.Debug().
				Str("path", targetFilePath).
				Str("library", lib.Name).
				Uint64("target_size", targetFile.Size).
				Uint64("local_size", s.Size).
				Msg("Local file size does not match in target datastore")

			process = true
			break
		}
	}

	if !process {
		// all local files existed in target
		t.log.Debug().
			Interface("scans", scans).
			Msg("All local files exist in target")
		return nil
	}

	// send scan request
	l := t.log.With().
		Str("path", scanFolder).
		Str("library", lib.Name).
		Logger()

	l.Trace().Msg("Sending scan request")

	if err := t.api.Scan(scanFolder); err != nil {
		return err
	}

	l.Info().Msg("Scan moved to target")
	return nil
}

func (t target) getScanLibrary(folder string) (*library, error) {
	for _, l := range t.libraries {
		if strings.HasPrefix(folder, l.Path) {
			return &l, nil
		}
	}

	return nil, fmt.Errorf("%v: failed determining library", folder)
}
