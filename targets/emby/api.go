package emby

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog"
)

type apiClient struct {
	url   string
	token string

	client *http.Client
	log    zerolog.Logger
}

func newAPIClient(c Config) *apiClient {
	return &apiClient{
		client: &http.Client{},
		url:    c.URL,
		token:  c.Token,
		log: autoscan.GetLogger(c.Verbosity).With().
			Str("target", "emby").
			Str("url", c.URL).
			Logger(),
	}
}

func (c apiClient) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Token", c.token)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, nil
	}

	// statusCode not in the 2xx range, close response
	res.Body.Close()

	switch res.StatusCode {
	case 401:
		return nil, fmt.Errorf("invalid emby token: %s: %w", res.Status, autoscan.ErrFatal)
	case 500, 503, 504:
		return nil, fmt.Errorf("%s: %w", res.Status, autoscan.ErrTargetUnavailable)
	default:
		return nil, fmt.Errorf("%s: %w", res.Status, autoscan.ErrFatal)
	}
}

func (c apiClient) Available() error {
	// create request
	reqURL := autoscan.JoinURL(c.url, "emby", "System", "Info")
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed creating availability request: %v: %w", err, autoscan.ErrFatal)
	}

	// send request
	res, err := c.do(req)
	if err != nil {
		return fmt.Errorf("availability: %w", err)
	}

	defer res.Body.Close()
	return nil
}

type library struct {
	Name string
	Path string
}

func (c apiClient) Libraries() ([]library, error) {
	// create request
	reqURL := autoscan.JoinURL(c.url, "emby", "Library", "SelectableMediaFolders")
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating libraries request: %v: %w", err, autoscan.ErrFatal)
	}

	// send request
	res, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("libraries: %w", err)
	}

	defer res.Body.Close()

	// decode response
	type Response struct {
		Name    string `json:"Name"`
		Folders []struct {
			Path string `json:"Path"`
		} `json:"SubFolders"`
	}

	resp := make([]Response, 0)
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed decoding libraries request response: %v: %w", err, autoscan.ErrFatal)
	}

	// process response
	libraries := make([]library, 0)
	for _, lib := range resp {
		for _, folder := range lib.Folders {
			libraries = append(libraries, library{
				Name: lib.Name,
				Path: folder.Path,
			})
		}
	}

	return libraries, nil
}

type scanRequest struct {
	Path       string `json:"path"`
	UpdateType string `json:"updateType"`
}

func (c apiClient) Scan(path string) error {
	// create request payload
	type Payload struct {
		Updates []scanRequest `json:"Updates"`
	}

	payload := &Payload{
		Updates: []scanRequest{
			{
				Path:       path,
				UpdateType: "Created",
			},
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed encoding scan request payload: %v: %w", err, autoscan.ErrFatal)
	}

	// create request
	reqURL := autoscan.JoinURL(c.url, "Library", "Media", "Updated")
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("failed creating scan request: %v: %w", err, autoscan.ErrFatal)
	}

	// send request
	res, err := c.do(req)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	defer res.Body.Close()
	return nil
}
