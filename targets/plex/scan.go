package plex

import (
	"fmt"
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

	libs, err := t.getScanLibrary(scanFolder)
	if err != nil {
		t.log.Warn().
			Err(err).
			Msg("No target libraries found")
		return fmt.Errorf("%v: %w", err, autoscan.ErrRetryScan)
	}

	// send scan request
	for _, lib := range libs {
		l := t.log.With().
			Str("path", scanFolder).
			Str("library", lib.Name).
			Logger()

		l.Trace().Msg("Sending scan request")

		if err := t.api.Scan(scanFolder, lib.ID); err != nil {
			return err
		}

		l.Info().Msg("Scan moved to target")
	}

	return nil
}

func (t target) getScanLibrary(folder string) ([]library, error) {
	libraries := make([]library, 0)

	for _, l := range t.libraries {
		if strings.HasPrefix(folder, l.Path) {
			libraries = append(libraries, l)
		}
	}

	if len(libraries) == 0 {
		return nil, fmt.Errorf("%v: failed determining libraries", folder)
	}

	return libraries, nil
}
