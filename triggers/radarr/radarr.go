package radarr

import (
	"encoding/json"
	"net/http"
	"path"
	"strconv"

	"github.com/cloudbox/autoscan"
)

// New creates an autoscan-compatible HTTP Trigger for Radarr webhooks.
//
// TODO: New should accept a Radarr-specific path rewriter.
func New() autoscan.HTTPTrigger {
	return func(scans chan autoscan.Scan) http.Handler {
		return &handler{scans: scans}
	}
}

type handler struct {
	scans chan autoscan.Scan
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

	// TODO: Rewrite the path based on the path-rewriter in the handler struct.
	scan := autoscan.Scan{
		Path:     path.Join(event.Movie.FolderPath, event.File.RelativePath),
		Priority: 1,
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
