package plex

import "github.com/cloudbox/autoscan"

func (t Target) Scan(scans []autoscan.Scan) error {
	t.log.Info().
		Msgf("Scanning: %+v", scans)
	return nil
}
