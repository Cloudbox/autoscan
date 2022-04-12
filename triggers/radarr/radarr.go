package radarr

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog/hlog"

	"github.com/cloudbox/autoscan"
)

type Config struct {
	Name      string             `yaml:"name"`
	Priority  int                `yaml:"priority"`
	Rewrite   []autoscan.Rewrite `yaml:"rewrite"`
	Verbosity string             `yaml:"verbosity"`
}

// New creates an autoscan-compatible HTTP Trigger for Radarr webhooks.
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

type radarrEvent struct {
	Type string `json:"eventType"`

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

	if strings.EqualFold(event.Type, "Test") {
		rlog.Info().Msg("Received test event")
		rw.WriteHeader(http.StatusOK)
		return
	}

	var folderPath string

	if strings.EqualFold(event.Type, "Download") || strings.EqualFold(event.Type, "MovieFileDelete") {
		if event.File.RelativePath == "" || event.Movie.FolderPath == "" {
			rlog.Error().Msg("Required fields are missing")
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		folderPath = path.Dir(path.Join(event.Movie.FolderPath, event.File.RelativePath))
	}

	if strings.EqualFold(event.Type, "MovieDelete") || strings.EqualFold(event.Type, "Rename") {
		if event.Movie.FolderPath == "" {
			rlog.Error().Msg("Required fields are missing")
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		folderPath = event.Movie.FolderPath
	}

	scan := autoscan.Scan{
		Folder:   h.rewrite(folderPath),
		Priority: h.priority,
		Time:     now(),
	}

	err = h.callback(scan)
	if err != nil {
		rlog.Error().Err(err).Msg("Processor could not process scan")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rlog.Info().
		Str("path", folderPath).
		Str("event", event.Type).
		Msg("Scan moved to processor")

	rw.WriteHeader(http.StatusOK)
}

var now = time.Now
