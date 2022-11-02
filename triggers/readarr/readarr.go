package readarr

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

// New creates an autoscan-compatible HTTP Trigger for Readarr webhooks.
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

type readarrEvent struct {
	Type    string `json:"eventType"`
	Upgrade bool   `json:"isUpgrade"`

	Files []struct {
		Path string
	} `json:"bookFiles"`
}

func (h handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var err error
	l := hlog.FromRequest(r)

	event := new(readarrEvent)
	err = json.NewDecoder(r.Body).Decode(event)
	if err != nil {
		l.Error().Err(err).Msg("Failed decoding request")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	l.Trace().Interface("event", event).Msg("Received JSON body")

	if strings.EqualFold(event.Type, "Test") {
		l.Info().Msg("Received test event")
		rw.WriteHeader(http.StatusOK)
		return
	}

	//Only handle test and download. Everything else is ignored.
	if !strings.EqualFold(event.Type, "Download") || len(event.Files) == 0 {
		l.Error().Msg("Required fields are missing")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	var unique = map[string]bool{}
	var scans []autoscan.Scan

	for _, f := range event.Files {
		folderPath := path.Dir(h.rewrite(f.Path))
		if _, ok := unique[folderPath]; ok {
			continue
		}

		// add scan
		unique[folderPath] = true
		scan := autoscan.Scan{
			Folder:   folderPath,
			Priority: h.priority,
			Time:     now(),
		}

		scans = append(scans, scan)
	}

	err = h.callback(scans...)
	if err != nil {
		l.Error().Err(err).Msg("Processor could not process scans")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, scan := range scans {
		l.Info().
			Str("path", scan.Folder).
			Str("event", event.Type).
			Msg("Scan moved to processor")
	}

	rw.WriteHeader(http.StatusOK)
}

var now = time.Now
