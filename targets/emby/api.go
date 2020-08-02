package emby

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog"
	"net/http"
	"net/url"
	"strings"
)

type apiClient struct {
	url   string
	token string
	log   zerolog.Logger
}

func newApiClient(c Config) *apiClient {
	return &apiClient{
		url:   c.URL,
		token: c.Token,
		log: autoscan.GetLogger(c.Verbosity).With().
			Str("target", "emby").
			Str("url", c.URL).
			Logger(),
	}
}

func (c apiClient) Available() error {
	// create request
	req, err := http.NewRequest("GET", autoscan.JoinURL(c.url, "emby", "System", "Info"), nil)
	if err != nil {
		return fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("X-Emby-Token", c.token)
	req.Header.Set("Accept", "application/json")

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not check Emby availability: %v: %w",
			err, autoscan.ErrTargetUnavailable)
	}

	defer res.Body.Close()

	// validate response
	if res.StatusCode != 200 {
		return fmt.Errorf("could not check Emby availability: %v: %w",
			res.StatusCode, autoscan.ErrTargetUnavailable)
	}

	return nil
}

type library struct {
	ID   string
	Name string
	Path string
}

func (c apiClient) Libraries() ([]library, error) {
	// create request
	req, err := http.NewRequest("GET",
		autoscan.JoinURL(c.url, "emby", "Library", "SelectableMediaFolders"), nil)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("X-Emby-Token", c.token)
	req.Header.Set("Accept", "application/json")

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve Emby libraries: %v: %w",
			err, autoscan.ErrTargetUnavailable)
	}

	defer res.Body.Close()

	// validate response
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("could not retrieve Emby libraries: %v: %w",
			res.StatusCode, autoscan.ErrTargetUnavailable)
	}

	// decode response
	resp := make([]struct {
		Name    string `json:"Name"`
		ID      string `json:"Id"`
		Folders []struct {
			Path string `json:"Path"`
		} `json:"SubFolders"`
	}, 0)

	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("could not decode Emby library response: %v: %w", err, autoscan.ErrTargetUnavailable)
	}

	// process response
	libraries := make([]library, 0)
	for _, lib := range resp {
		for _, folder := range lib.Folders {
			libraries = append(libraries, library{
				Name: lib.Name,
				ID:   lib.ID,
				Path: folder.Path,
			})
		}
	}

	return libraries, nil
}

type mediaPart struct {
	File string
	Size uint64
}

func (c apiClient) MediaPartByFile(libraryId string, path string) (*mediaPart, error) {
	// create request
	req, err := http.NewRequest("GET", autoscan.JoinURL(c.url, "emby", "Items"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating search items request: %v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("X-Emby-Token", c.token)
	req.Header.Set("Accept", "application/json")

	// set params
	q := url.Values{}
	q.Add("Path", path)
	q.Add("ParentId", libraryId)
	q.Add("Fields", "MediaSources")

	req.URL.RawQuery = q.Encode()

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed sending search items request: %v: %w", err, autoscan.ErrRetryScan)
	}

	defer res.Body.Close()

	// validate response
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%v: failed validating search items request response: %w",
			res.Status, autoscan.ErrRetryScan)
	}

	// decode response
	resp := new(struct {
		Items []struct {
			Name         string `json:"Name"`
			MediaSources []struct {
				Path string `json:"Path"`
				Size uint64 `json:"Size"`
			} `json:"MediaSources"`
		} `json:"Items"`
	})

	if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
		return nil, fmt.Errorf("failed decoding search items request response: %v: %w", err, autoscan.ErrFatal)
	}

	// process response
	for _, item := range resp.Items {
		for _, file := range item.MediaSources {
			if strings.EqualFold(file.Path, path) {
				return &mediaPart{
					File: file.Path,
					Size: file.Size,
				}, nil
			}
		}
	}

	return nil, autoscan.ErrNotFoundInTarget
}

type scanRequest struct {
	Path       string `json:"path"`
	UpdateType string `json:"updateType"`
}

func (c apiClient) Scan(path string) error {
	// create request payload
	payload := new(struct {
		Updates []scanRequest `json:"Updates"`
	})

	payload.Updates = append(payload.Updates, scanRequest{
		Path:       path,
		UpdateType: "Created",
	})

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed encoding scan request payload: %v, %w", err, autoscan.ErrFatal)
	}

	// create request
	reqURL := autoscan.JoinURL(c.url, "Library", "Media", "Updated")
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(b))
	if err != nil {
		// May only occur when the user has provided an invalid URL
		return fmt.Errorf("failed creating scan request: %v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Token", c.token)

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed sending scan request: %v: %w", err, autoscan.ErrTargetUnavailable)
	}

	defer res.Body.Close()

	// validate response
	if res.StatusCode != 204 {
		// 404 if some kind of proxy is in-front of Emby while it's offline.
		return fmt.Errorf("%v: failed validating scan request response: %w", res.Status, autoscan.ErrTargetUnavailable)
	}

	return nil
}
