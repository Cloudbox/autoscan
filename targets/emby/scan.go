package emby

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

	lib, err := t.getScanLibrary(scanFolder)
	if err != nil {
		t.log.Warn().
			Err(err).
			Msg("No target library found")
		return fmt.Errorf("%v: %w", err, autoscan.ErrRetryScan)
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
