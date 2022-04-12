package manual

import (
	_ "embed"
	"net/http"
	"path"
	"time"

	"github.com/rs/zerolog/hlog"

	"github.com/cloudbox/autoscan"
)

type Config struct {
	Rewrite   []autoscan.Rewrite `yaml:"rewrite"`
	Priority  int                `yaml:"priority"`
	Verbosity string             `yaml:"verbosity"`
}

var (
	//go:embed "template.html"
	template []byte
)

// New creates an autoscan-compatible HTTP Trigger for manual webhooks.
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

func (h handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var err error
	rlog := hlog.FromRequest(r)

	query := r.URL.Query()
	directories := query["dir"]

	switch r.Method {
	case "GET":
		rw.Header().Set("Content-Type", "text/html")
		_, _ = rw.Write(template)
		return
	case "HEAD":
		rw.WriteHeader(http.StatusOK)
		return
	}

	if len(directories) == 0 {
		rlog.Error().Msg("Manual webhook should receive at least one directory")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rlog.Trace().Interface("dirs", directories).Msg("Received directories")

	scans := make([]autoscan.Scan, 0)

	for _, dir := range directories {
		// Rewrite the path based on the provided rewriter.
		folderPath := h.rewrite(path.Clean(dir))

		scans = append(scans, autoscan.Scan{
			Folder:   folderPath,
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

	rw.WriteHeader(http.StatusOK)
	for _, scan := range scans {
		rlog.Info().
			Str("path", scan.Folder).
			Msg("Scan moved to processor")
	}
}

var now = time.Now
