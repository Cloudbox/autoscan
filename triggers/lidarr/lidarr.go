package lidarr

import (
	"encoding/json"
	"net/http"
	"path"
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

	dirMap := make(map[string]int)
	folderPaths := make([]string, 0)
	scans := make([]autoscan.Scan, 0)

	for _, f := range event.Files {
		folderPath := path.Dir(h.rewrite(f.Path))
		if _, ok := dirMap[folderPath]; ok {
			continue
		}
		dirMap[folderPath] = 1

		// add scan
		folderPaths = append(folderPaths, folderPath)
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
		Strs("path", folderPaths).
		Msg("Scan moved to processor")
}

var now = time.Now
