package lidarr

import (
	"encoding/json"
	"net/http"
	"path"

	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog/hlog"
)

type Config struct {
	Name      string             `yaml:"name"`
	Priority  int                `yaml:"priority"`
	Rewrite   []autoscan.Rewrite `yaml:"rewrite"`
	Verbosity string             `yaml:"verbosity"`
}

// New creates an autoscan-compatible HTTP Trigger for Lidarr webhooks.
func New(c Config) (autoscan.HTTPTrigger, error) {
	rewriter, err := autoscan.NewRewriter(c.Rewrite)
	if err != nil {
		return nil, err
	}

	trigger := func(callback autoscan.ProcessorFunc) http.Handler {
		return handler{
			callback: callback,
			priority: c.Priority,
			rewrite:  rewriter,
		}
	}

	return trigger, nil
}

type handler struct {
	priority int
	rewrite  autoscan.Rewriter
	callback autoscan.ProcessorFunc
}

type lidarrEvent struct {
	Type    string `json:"eventType"`
	Upgrade bool   `json:"isUpgrade"`

	Files []struct {
		Path string
	} `json:"trackFiles"`

	Artist struct {
		Name string
		Path string
	} `json:"artist"`
}

func (h handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var err error
	l := hlog.FromRequest(r)

	event := new(lidarrEvent)
	err = json.NewDecoder(r.Body).Decode(event)
	if err != nil {
		l.Error().Err(err).Msg("Failed decoding request")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	l.Trace().Interface("event", event).Msg("Received JSON body")

	if event.Type == "Test" {
		l.Debug().Msg("Received test event")
		rw.WriteHeader(http.StatusOK)
		return
	}

	if event.Type != "Download" || len(event.Files) == 0 {
		l.Error().Msg("Required fields are missing")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	var scans []autoscan.Scan

	for _, f := range event.Files {
		// Rewrite the path based on the provided rewriter.
		fullPath := h.rewrite(f.Path)

		scans = append(scans, autoscan.Scan{
			File:     path.Base(fullPath),
			Folder:   path.Dir(fullPath),
			Priority: h.priority,
			Removed:  false,
		})
	}

	err = h.callback(scans...)
	if err != nil {
		l.Error().Err(err).Msg("Processor could not process scans")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	l.Info().
		Str("path", h.rewrite(event.Artist.Path)).
		Int("files", len(scans)).
		Msg("Scan moved to processor")
}
