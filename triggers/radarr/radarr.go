package radarr

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog/hlog"
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
	Type    string `json:"eventType"`
	Upgrade bool   `json:"isUpgrade"`

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
		rlog.Debug().Msg("Received test event")
		rw.WriteHeader(http.StatusOK)
		return
	}

	if !strings.EqualFold(event.Type, "Download") || event.File.RelativePath == "" || event.Movie.FolderPath == "" {
		rlog.Error().Msg("Required fields are missing")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Rewrite the path based on the provided rewriter.
	folderPath := path.Dir(h.rewrite(path.Join(event.Movie.FolderPath, event.File.RelativePath)))

	scan := autoscan.Scan{
		Folder:   folderPath,
		Priority: h.priority,
		Time:     now(),
	}

	err = h.callback(scan)
	if err != nil {
		rlog.Error().Err(err).Msg("Processor could not process scan")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rlog.Info().
		Str("path", folderPath).
		Msg("Scan moved to processor")
}

var now = time.Now
