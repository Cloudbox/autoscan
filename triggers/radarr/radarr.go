package radarr

import (
	"encoding/json"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/cloudbox/autoscan"
)

type Config struct {
	Name     string           `yaml:"name"`
	Priority int              `yaml:"priority"`
	Rewrite  autoscan.Rewrite `yaml:"rewrite"`
}

// New creates an autoscan-compatible HTTP Trigger for Radarr webhooks.
func New(c Config) (trigger autoscan.HTTPTrigger, err error) {
	rewriter, err := autoscan.NewRewriter(c.Rewrite)
	if err != nil {
		return
	}

	trigger = func(callback autoscan.ProcessorFunc) http.Handler {
		return &handler{
			callback: callback,
			priority: c.Priority,
			rewrite:  rewriter,
			log: log.With().
				Str("trigger", c.Name).
				Logger(),
		}
	}

	return
}

type handler struct {
	priority int
	rewrite  autoscan.Rewriter
	callback autoscan.ProcessorFunc

	log zerolog.Logger
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

	rlog := h.log.With().
		Str("remote", r.RemoteAddr).
		Logger()

	event := new(radarrEvent)
	err = json.NewDecoder(r.Body).Decode(event)
	if err != nil {
		rlog.Error().
			Err(err).
			Msg("Failed decoding request")

		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rlog.Debug().
		Interface("event", event).
		Msg("Processing request")

	if event.Type == "Test" {
		return
	}

	if event.Type != "Download" || event.File.RelativePath == "" || event.Movie.FolderPath == "" {
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
			Str("trigger_path", fullPath).
			Msg("Failed determining trigger file size")

		rw.WriteHeader(404)
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
		rlog.Error().
			Err(err).
			Msg("Failed processing request")

		rw.WriteHeader(500)
		return
	}

	rlog.Info().
		Str("trigger_path", fullPath).
		Msg("Request processed")
}

var fileSize = func(name string) (uint64, error) {
	info, err := os.Stat(name)
	if err != nil {
		return 0, err
	}

	return uint64(info.Size()), nil
}
