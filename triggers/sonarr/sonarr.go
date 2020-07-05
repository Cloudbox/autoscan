package sonarr

import (
	"encoding/json"
	"net/http"
	"path"
	"strconv"

	"github.com/cloudbox/autoscan"
)

// New creates an autoscan-compatible HTTP Trigger for Sonarr webhooks.
func New(c Config) autoscan.HTTPTrigger {
	return func(scans chan autoscan.Scan) http.Handler {
		return &handler{config: c, scans: scans}
	}
}

type Config struct {
	autoscan.HTTPTriggerConfig `yaml:",inline"`
}

type handler struct {
	config Config
	scans  chan autoscan.Scan
}

type sonarrEvent struct {
	Type    string `json:"eventType"`
	Upgrade bool   `json:"isUpgrade"`

	File struct {
		RelativePath string
	} `json:"episodeFile"`

	Series struct {
		Title  string
		Path   string
		TvdbID int
	} `json:"series"`
}

func (h handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var err error

	event := new(sonarrEvent)
	err = json.NewDecoder(r.Body).Decode(event)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if event.Type == "Test" {
		return
	}

	if event.Type != "Download" || event.File.RelativePath == "" || event.Series.Path == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Rewrite the path based on the provided AddPrefix and StripPrefix values.
	filePath := path.Join(event.Series.Path, event.File.RelativePath)
	if h.config.StripPrefix != "" {
		filePath = autoscan.StripPrefix(h.config.StripPrefix, filePath)
	}

	if h.config.AddPrefix != "" {
		filePath = autoscan.AddPrefix(h.config.AddPrefix, filePath)
	}

	scan := autoscan.Scan{
		Path:     filePath,
		Priority: h.config.Priority,
	}

	if event.Series.TvdbID != 0 {
		scan.Metadata.Provider = autoscan.TVDb
		scan.Metadata.ID = strconv.Itoa(event.Series.TvdbID)
	}

	h.scans <- scan
}
