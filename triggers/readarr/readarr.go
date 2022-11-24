package readarr

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

	unique := make(map[string]bool)
	scans := make([]autoscan.Scan, 0)

	for _, f := range event.Files {
		folderPath := path.Dir(h.rewrite(f.Path))
		if _, ok := unique[folderPath]; ok {
			continue
		}

		// add scan
		unique[folderPath] = true
		scans = append(scans, autoscan.Scan{
			Folder:   folderPath,
			Priority: h.priority,
			Time:     now(),
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
		Str("path", scans[0].Folder).
		Str("event", event.Type).
		Msg("Scan moved to processor")
}

var now = time.Now
