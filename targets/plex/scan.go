package plex

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudbox/autoscan"
)

func (t target) Scan(scans []autoscan.Scan) error {
	// ensure scan tasks present (should never fail)
	if len(scans) == 0 {
		return nil
	}

	// check for at-least one missing/changed file
	process := false
	for _, s := range scans {
		fp := t.rewrite(filepath.Join(s.Folder, s.File))

		pf, err := t.store.MediaPartByFile(fp)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// trigger file not found in target
				t.log.Debug().
					Str("target_path", fp).
					Msg("At least one scan does not exist in target")

				process = true
				break
			}

			// unexpected error
			return fmt.Errorf("could not check plex datastore: %v: %w", err, autoscan.ErrFatal)
		}

		// trigger file was found in target
		if pf.Size != s.Size {
			// trigger file did not match in target
			t.log.Debug().
				Str("target_path", fp).
				Uint64("target_size", pf.Size).
				Uint64("trigger_size", s.Size).
				Msg("Trigger file size does not match target file")

			process = true
			break
		}
	}

	if !process {
		// all scan task files existed in target
		t.log.Debug().
			Interface("scans", scans).
			Msg("All trigger files existed within target")
		return nil
	}

	s := scans[0]
	scanFolder := t.rewrite(s.Folder)

	// determine library for this scan
	lib, err := t.getScanLibrary(scanFolder)
	if err != nil {
		t.log.Warn().
			Err(err).
			Int("target_retries", s.Retries).
			Msg("No target library found")
		return fmt.Errorf("%v: %w", err, autoscan.ErrRetryScan)
	}

	slog := t.log.With().
		Str("target_path", scanFolder).
		Str("target_library", lib.Name).
		Int("target_retries", s.Retries).
		Logger()

	slog.Debug().Msg("Sending scan request")

	// create request
	reqURL := autoscan.JoinURL(t.url, "library", "sections", strconv.Itoa(lib.ID), "refresh")
	req, err := http.NewRequest("PUT", reqURL, nil)
	if err != nil {
		// May only occur when the user has provided an invalid URL
		return fmt.Errorf("failed creating scan request: %v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("X-Plex-Token", t.token)

	// set params
	q := url.Values{}
	q.Add("path", scanFolder)

	req.URL.RawQuery = q.Encode()

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed sending scan request: %v: %w", err, autoscan.ErrTargetUnavailable)
	}

	defer res.Body.Close()

	// validate response
	if res.StatusCode != 200 {
		// 404 if some kind of proxy is in-front of Plex while it's offline.
		return fmt.Errorf("%v: failed validating scan request response: %w", res.Status, autoscan.ErrTargetUnavailable)
	}

	slog.Info().Msg("Scan requested")
	return nil
}

func (t target) getScanLibrary(folder string) (*Library, error) {
	for _, l := range t.libraries {
		if strings.HasPrefix(folder, l.Path) {
			return &l, nil
		}
	}

	return nil, fmt.Errorf("%v: failed determining library", folder)
}
