package plex

import (
	"errors"
	"fmt"
	"github.com/cloudbox/autoscan"
	"path/filepath"
)

var ErrScanner = errors.New("scanner: scanner related error")

func (t Target) Scan(scans []autoscan.Scan) error {
	// ensure scan tasks present (should never fail)
	if len(scans) == 0 {
		return fmt.Errorf("no scan tasks present: %w", ErrScanner)
	}

	// check for at-least one missing/changed file
	continueScan := false
	for _, s := range scans {
		fp := filepath.Join(s.Folder, s.File)

		pf, err := t.store.MediaPartByFile(fp)
		if err != nil {
			if errors.Is(err, ErrDatabaseRowNotFound) {
				// trigger file not found in target
				t.log.Debug().
					Str("path", fp).
					Msg("Trigger file does not exist in target")

				continueScan = true
				break
			}

			// unexpected error, check the next file
			t.log.Error().
				Err(err).
				Str("path", fp).
				Msg("Failed checking if trigger file existed in target")

			continue
		}

		// trigger file was found in target
		if pf.Size != s.Size {
			// trigger file did not match in target
			t.log.Debug().
				Str("path", fp).
				Int64("target_size", pf.Size).
				Int64("trigger_size", s.Size).
				Msg("Trigger file size does not match targets file")

			continueScan = true
			break
		}
	}

	if !continueScan {
		// all the scan task files existed in target
		t.log.Debug().
			Msgf("All trigger files existed within target, skipping for: %+v", scans)
		return nil
	}

	s := scans[0]

	// scan folder
	t.log.Info().
		Str("target_path", s.Folder).
		Msg("Scanning")

	return nil
}
