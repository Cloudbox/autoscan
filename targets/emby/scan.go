package emby

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cloudbox/autoscan"
)

type scanRequest struct {
	Path       string `json:"path"`
	UpdateType string `json:"updateType"`
}

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
					Str("path", fp).
					Msg("At least one trigger file did not exist in target datastore")

				process = true
				break
			}

			// unexpected error
			return fmt.Errorf("could not check emby datastore: %v: %w", err, autoscan.ErrFatal)
		}

		// trigger file was found in target
		if pf.Size != s.Size {
			// trigger file did not match in target
			t.log.Debug().
				Str("path", fp).
				Uint64("target_size", pf.Size).
				Uint64("trigger_size", s.Size).
				Msg("Trigger file size does not match in target datastore")

			process = true
			break
		}
	}

	if !process {
		// all scan task files existed in target
		t.log.Debug().
			Interface("scans", scans).
			Msg("All trigger files existed in target")
		return nil
	}

	s := scans[0]
	scanFolder := t.rewrite(s.Folder)

	// determine library for this scan
	lib, err := t.getScanLibrary(scanFolder)
	if err != nil {
		t.log.Warn().
			Err(err).
			Int("retries", s.Retries).
			Msg("No target library found")
		return fmt.Errorf("%v: %w", err, autoscan.ErrRetryScan)
	}

	slog := t.log.With().
		Str("path", scanFolder).
		Str("library", lib.Name).
		Int("retries", s.Retries).
		Logger()

	slog.Debug().Msg("Sending scan request")

	// create request payload
	payload := new(struct {
		Updates []scanRequest `json:"Updates"`
	})

	payload.Updates = append(payload.Updates, scanRequest{
		Path:       scanFolder,
		UpdateType: "Created",
	})

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed encoding scan request payload: %v, %w", err, autoscan.ErrFatal)
	}

	// create request
	reqURL := autoscan.JoinURL(t.url, "Library", "Media", "Updated")
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(b))
	if err != nil {
		// May only occur when the user has provided an invalid URL
		return fmt.Errorf("failed creating scan request: %v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Token", t.token)

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed sending scan request: %v: %w", err, autoscan.ErrTargetUnavailable)
	}

	defer res.Body.Close()

	// validate response
	if res.StatusCode != 204 {
		// 404 if some kind of proxy is in-front of Emby while it's offline.
		return fmt.Errorf("%v: failed validating scan request response: %w", res.Status, autoscan.ErrTargetUnavailable)
	}

	slog.Info().Msg("Scan queued")
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
