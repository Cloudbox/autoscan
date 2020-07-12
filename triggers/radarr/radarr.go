package radarr

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/cloudbox/autoscan"
)

type Config struct {
	Name     string           `yaml:"name"`
	Priority int              `yaml:"priority"`
	Rewrite  autoscan.Rewrite `yaml:"rewrite"`
}

// New creates an autoscan-compatible HTTP Trigger for Radarr webhooks.
func New(c Config) (trigger autoscan.HTTPTrigger, err error) {
	rewriter, err := autoscan.NewRewriter(c.Rewrite)
	if err != nil {
		return
	}

	trigger = func(callback autoscan.ProcessorFunc) http.Handler {
		return &handler{
			callback: callback,
			priority: c.Priority,
			rewrite:  rewriter,
		}
	}

	return
}

type handler struct {
	priority int
	rewrite  autoscan.Rewriter
	callback autoscan.ProcessorFunc
}

type radarrEvent struct {
	Type    string `json:"eventType"`
	Upgrade bool   `json:"isUpgrade"`

	Details struct {
		TmdbID int
		ImdbID string
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

	// Rewrite the path based on the provided rewriter.
	fullPath := h.rewrite(path.Join(event.Movie.FolderPath, event.File.RelativePath))

	// Retrieve the size of the file.
	size, err := fileSize(fullPath)
	if err != nil {
		rw.WriteHeader(404)
		return
	}

	scan := autoscan.Scan{
		File:     path.Base(fullPath),
		Folder:   path.Dir(fullPath),
		Priority: h.priority,
		Size:     size,
	}

	if event.Details.ImdbID != "" {
		scan.Metadata.Provider = autoscan.IMDb
		scan.Metadata.ID = event.Details.ImdbID
	} else if event.Details.TmdbID != 0 {
		scan.Metadata.Provider = autoscan.TMDb
		scan.Metadata.ID = strconv.Itoa(event.Details.TmdbID)
	}

	err = h.callback(scan)
	if err != nil {
		rw.WriteHeader(500)
		return
	}
}

var fileSize func(string) (int64, error)

func init() {
	fileSize = func(name string) (int64, error) {
		info, err := os.Stat(name)
		if err != nil {
			return 0, err
		}

		return info.Size(), nil
	}
}
