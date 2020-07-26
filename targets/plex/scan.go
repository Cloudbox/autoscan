package plex

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudbox/autoscan"
)

var (
	ErrTargetUnexpected = errors.New("target: unexpected error")
	ErrTargetDatabase   = errors.New("target: database related error")
	ErrTargetRequest    = errors.New("target: request related error")
)

func (t target) Scan(scans []autoscan.Scan) error {
	// ensure scan tasks present (should never fail)
	if len(scans) == 0 {
		return fmt.Errorf("no scan tasks present: %w", ErrTargetUnexpected)
	}

	// check for at-least one missing/changed file
	process := false
	for _, s := range scans {
		fp := t.rewrite(filepath.Join(s.Folder, s.File))

		pf, err := t.store.MediaPartByFile(fp)
		if err != nil {
			if errors.Is(err, ErrDatabaseRowNotFound) {
				// trigger file not found in target
				t.log.Debug().
					Str("target_path", fp).
					Msg("Trigger file does not exist in target")

				process = true
				break
			}

			// unexpected error, check the next file
			t.log.Error().
				Err(err).
				Str("target_path", fp).
				Msg("Failed checking if trigger file existed in target")

			continue
		}

		// trigger file was found in target
		if pf.Size != s.Size {
			// trigger file did not match in target
			t.log.Debug().
				Str("target_path", fp).
				Uint64("target_size", pf.Size).
				Uint64("trigger_size", s.Size).
				Msg("Trigger file size does not match targets file")

			process = true
			break
		}
	}

	if !process {
		// all scan task files existed in target
		t.log.Debug().
			Msgf("All trigger files existed within target, skipping for: %+v", scans)
		return nil
	}

	s := scans[0]
	scanFolder := t.rewrite(s.Folder)

	// determine library for this scan
	lib, err := t.getScanLibrary(&s)
	if err != nil {
		t.log.Error().
			Err(err).
			Str("target_path", scanFolder).
			Msg("Failed determining target library to scan")
		return err
	}

	slog := t.log.With().
		Str("target_path", scanFolder).
		Str("target_library", lib.Name).
		Logger()

	slog.Debug().
		Msg("Sending scan request")

	// create request
	reqURL := autoscan.JoinURL(t.url, "library", "sections", strconv.Itoa(lib.ID), "refresh")
	req, err := http.NewRequest("PUT", reqURL, nil)
	if err != nil {
		slog.Error().
			Err(err).
			Msg("Failed creating scan request")
		return fmt.Errorf("failed creating scan request: %w", ErrTargetRequest)
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
		slog.Error().
			Err(err).
			Msg("Failed sending scan request")
		return fmt.Errorf("failed sending scan request: %w", ErrTargetRequest)
	}

	defer res.Body.Close()

	// validate response
	if res.StatusCode != 200 {
		slog.Error().
			Str("status", res.Status).
			Msg("Failed validating scan request response")
		return fmt.Errorf("failed validating scan request response: %w", ErrTargetRequest)
	}

	slog.Info().
		Msg("Scan has been requested")
	return nil
}

func (t target) getScanLibrary(scan *autoscan.Scan) (*Library, error) {
	for _, l := range t.libraries {
		if strings.HasPrefix(t.rewrite(scan.Folder), l.Path) {
			return &l, nil
		}
	}

	return nil, fmt.Errorf("failed determining library: %w", ErrTargetDatabase)
}