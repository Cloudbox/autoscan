package radarr

import (
	"encoding/json"
	"net/http"
	"path"
	"strconv"

	"github.com/cloudbox/autoscan"
)

// New creates an autoscan-compatible HTTP Trigger for Radarr webhooks.
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

type radarrEvent struct {
	Type    string `json:"eventType"`
	Upgrade bool   `json:"isUpgrade"`

	Details struct {
		TmdbID int
		ImdbID string
		Title  string
		Year   int
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

	event := new(radarrEvent)
	err = json.NewDecoder(r.Body).Decode(event)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if event.Type == "Test" {
		return
	}

	if event.Type != "Download" || event.File.RelativePath == "" || event.Movie.FolderPath == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Rewrite the path based on the provided AddPrefix and StripPrefix values.
	filePath := path.Join(event.Movie.FolderPath, event.File.RelativePath)
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

	if event.Details.ImdbID != "" {
		scan.Metadata.Provider = autoscan.IMDb
		scan.Metadata.ID = event.Details.ImdbID
	} else if event.Details.TmdbID != 0 {
		scan.Metadata.Provider = autoscan.TMDb
		scan.Metadata.ID = strconv.Itoa(event.Details.TmdbID)
	}

	h.scans <- scan
}
