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
		targetFilePath := t.rewrite(filepath.Join(s.Folder, s.File))

		targetFile, err := t.store.MediaPartByFile(targetFilePath)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// local file not found in target
				t.log.Debug().
					Str("path", targetFilePath).
					Msg("At least one local file did not exist in target datastore")

				process = true
				break
			}

			// unexpected error
			return fmt.Errorf("could not check plex datastore: %v: %w", err, autoscan.ErrFatal)
		}

		// local file was found in target
		if targetFile.Size != s.Size {
			// local file did not match in target
			t.log.Debug().
				Str("path", targetFilePath).
				Uint64("target_size", targetFile.Size).
				Uint64("local_size", s.Size).
				Msg("Local file size does not match in target datastore")

			process = true
			break
		}
	}

	if !process {
		// all local task files existed in target
		t.log.Debug().
			Interface("scans", scans).
			Msg("All local files exist in target")
		return nil
	}

	scanFolder := t.rewrite(scans[0].Folder)

	// determine library for this scan
	lib, err := t.getScanLibrary(scanFolder)
	if err != nil {
		t.log.Warn().
			Err(err).
			Msg("No target library found")
		return fmt.Errorf("%v: %w", err, autoscan.ErrRetryScan)
	}

	l := t.log.With().
		Str("path", scanFolder).
		Str("library", lib.Name).
		Logger()

	l.Trace().Msg("Sending scan request")

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

	l.Info().Msg("Scan queued")
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
