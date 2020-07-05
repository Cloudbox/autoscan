package sonarr

import (
	"encoding/json"
	"net/http"
	"path"
	"strconv"

	"github.com/cloudbox/autoscan"
)

// New creates an autoscan-compatible HTTP Trigger for Sonarr webhooks.
//
// TODO: New should accept a Sonarr-specific path rewriter.
func New() autoscan.HTTPTrigger {
	return func(scans chan autoscan.Scan) http.Handler {
		return &handler{scans: scans}
	}
}

type handler struct {
	scans chan autoscan.Scan
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

	// TODO: Rewrite the path based on the path-rewriter in the handler struct.
	scan := autoscan.Scan{
		Path:     path.Join(event.Series.Path, event.File.RelativePath),
		Priority: 1,
	}

	if event.Series.TvdbID != 0 {
		scan.Metadata.Provider = autoscan.TVDb
		scan.Metadata.ID = strconv.Itoa(event.Series.TvdbID)
	}

	h.scans <- scan
}
