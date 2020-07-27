package radarr

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/rs/zerolog/hlog"

	"github.com/cloudbox/autoscan"
	"github.com/cloudbox/autoscan/triggers"
)

type Config struct {
	Name     string           `yaml:"name"`
	Priority int              `yaml:"priority"`
	Rewrite  autoscan.Rewrite `yaml:"rewrite"`
}

// New creates an autoscan-compatible HTTP Trigger for Radarr webhooks.
func New(c Config) (autoscan.HTTPTrigger, error) {
	rewriter, err := autoscan.NewRewriter(c.Rewrite)
	if err != nil {
		return nil, err
	}

	trigger := func(callback autoscan.ProcessorFunc) http.Handler {
		return triggers.WithLogger(handler{
			callback: callback,
			priority: c.Priority,
			rewrite:  rewriter,
		})
	}

	return trigger, nil
}

type handler struct {
	priority int
	rewrite  autoscan.Rewriter
	callback autoscan.ProcessorFunc
}

type radarrEvent struct {
	Type    string `json:"eventType"`
	Upgrade bool   `json:"isUpgrade"`

	Details struct {
		TmdbID int
		ImdbID string
	} `json:"remoteMovie"`

	File struct {
		RelativePath string
	} `json:"movieFile"`

	Movie struct {
		FolderPath string
	} `json:"movie"`
}

func (h handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var err error
	rlog := hlog.FromRequest(r)

	event := new(radarrEvent)
	err = json.NewDecoder(r.Body).Decode(event)
	if err != nil {
		rlog.Error().Err(err).Msg("Failed decoding request")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rlog.Trace().Interface("event", event).Msg("Received JSON body")

	if event.Type == "Test" {
		rlog.Debug().Msg("Received test event")
		rw.WriteHeader(http.StatusOK)
		return
	}

	if event.Type != "Download" || event.File.RelativePath == "" || event.Movie.FolderPath == "" {
		rlog.Error().Msg("Required fields are missing")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Rewrite the path based on the provided rewriter.
	fullPath := h.rewrite(path.Join(event.Movie.FolderPath, event.File.RelativePath))

	// Retrieve the size of the file.
	size, err := fileSize(fullPath)
	if err != nil {
		rlog.Warn().
			Err(err).
			Str("path", fullPath).
			Msg("File does not exist")

		rw.WriteHeader(http.StatusNotFound)
		return
	}

	scan := autoscan.Scan{
		File:     path.Base(fullPath),
		Folder:   path.Dir(fullPath),
		Priority: h.priority,
		Size:     size,
	}

	if event.Details.ImdbID != "" {
		scan.Metadata.Provider = autoscan.IMDb
		scan.Metadata.ID = event.Details.ImdbID
	} else if event.Details.TmdbID != 0 {
		scan.Metadata.Provider = autoscan.TMDb
		scan.Metadata.ID = strconv.Itoa(event.Details.TmdbID)
	}

	err = h.callback(scan)
	if err != nil {
		rlog.Error().Err(err).Msg("Processor could not process scan")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rlog.Info().
		Str("path", fullPath).
		Msg("Scan queued")
}

var fileSize = func(name string) (uint64, error) {
	info, err := os.Stat(name)
	if err != nil {
		return 0, err
	}

	return uint64(info.Size()), nil
}
