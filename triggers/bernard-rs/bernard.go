package bernard_rs

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cloudbox/autoscan"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/hlog"
)

type Drive struct {
	ID      string             `yaml:"id"`
	Rewrite []autoscan.Rewrite `yaml:"rewrite"`
}

type Config struct {
	Drives    []Drive            `yaml:"drives"`
	Priority  int                `yaml:"priority"`
	Rewrite   []autoscan.Rewrite `yaml:"rewrite"`
	Verbosity string             `yaml:"verbosity"`
}

type BernardRewriter = func(drive string, input string) string

// // New creates an autoscan-compatible HTTP Trigger for Bernard (Rust edition) webhooks.
func New(c Config) (autoscan.HTTPTrigger, error) {
	rewrites := make(map[string]autoscan.Rewriter)
	for _, drive := range c.Drives {
		rewriter, err := autoscan.NewRewriter(append(drive.Rewrite, c.Rewrite...))
		if err != nil {
			return nil, err
		}

		rewrites[drive.ID] = rewriter
	}

	globalRewriter, err := autoscan.NewRewriter(c.Rewrite)
	if err != nil {
		return nil, err
	}

	rewriter := func(drive string, input string) string {
		driveRewriter, ok := rewrites[drive]
		if !ok {
			return globalRewriter(input)
		}

		return driveRewriter(input)
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
	rewrite  BernardRewriter
	callback autoscan.ProcessorFunc
}

type bernardEvent struct {
	Created []string
	Deleted []string
}

func (h handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var err error
	rlog := hlog.FromRequest(r)

	drive := chi.URLParam(r, "drive")

	event := new(bernardEvent)
	err = json.NewDecoder(r.Body).Decode(event)
	if err != nil {
		rlog.Error().Err(err).Msg("Failed decoding request")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rlog.Trace().Interface("event", event).Msg("Received JSON body")

	scans := make([]autoscan.Scan, 0)

	for _, path := range event.Created {
		scans = append(scans, autoscan.Scan{
			Folder:   h.rewrite(drive, path),
			Priority: h.priority,
			Time:     now(),
		})
	}

	for _, path := range event.Deleted {
		scans = append(scans, autoscan.Scan{
			Folder:   h.rewrite(drive, path),
			Priority: h.priority,
			Time:     now(),
		})
	}

	err = h.callback(scans...)
	if err != nil {
		rlog.Error().Err(err).Msg("Processor could not process scans")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, scan := range scans {
		rlog.Info().Str("path", scan.Folder).Msg("Scan moved to processor")
	}

	rw.WriteHeader(http.StatusOK)
}

var now = time.Now
