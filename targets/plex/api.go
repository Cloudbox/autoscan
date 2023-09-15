package plex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/cloudbox/autoscan"
)

type apiClient struct {
	client  *http.Client
	log     zerolog.Logger
	baseURL string
	token   string
}

func newAPIClient(baseURL string, token string, log zerolog.Logger) *apiClient {
	return &apiClient{
		client:  &http.Client{},
		log:     log,
		baseURL: baseURL,
		token:   token,
	}
}

func (c apiClient) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Plex-Token", c.token)
	req.Header.Set("Accept", "application/json") // Force JSON Response.

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, autoscan.ErrTargetUnavailable)
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, nil
	}

	c.log.Trace().
		Stringer("request_url", res.Request.URL).
		Int("response_status", res.StatusCode).
		Msg("Request failed")

	// statusCode not in the 2xx range, close response
	res.Body.Close()

	switch res.StatusCode {
	case 401:
		return nil, fmt.Errorf("invalid plex token: %s: %w", res.Status, autoscan.ErrFatal)
	case 404, 500, 502, 503, 504:
		return nil, fmt.Errorf("%s: %w", res.Status, autoscan.ErrTargetUnavailable)
	default:
		return nil, fmt.Errorf("%s: %w", res.Status, autoscan.ErrFatal)
	}
}

func (c apiClient) Version() (string, error) {
	reqURL := autoscan.JoinURL(c.baseURL)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed creating version request: %v: %w", err, autoscan.ErrFatal)
	}

	res, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("version: %w", err)
	}

	defer res.Body.Close()

	type Response struct {
		MediaContainer struct {
			Version string
		}
	}

	resp := new(Response)
	if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
		return "", fmt.Errorf("failed decoding version response: %v: %w", err, autoscan.ErrFatal)
	}

	return resp.MediaContainer.Version, nil
}

type library struct {
	ID   int
	Name string
	Path string
}

func (c apiClient) Libraries() ([]library, error) {
	reqURL := autoscan.JoinURL(c.baseURL, "library", "sections")
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating libraries request: %v: %w", err, autoscan.ErrFatal)
	}

	res, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("libraries: %w", err)
	}

	defer res.Body.Close()

	type Response struct {
		MediaContainer struct {
			Libraries []struct {
				ID       int    `json:"key,string"`
				Name     string `json:"title"`
				Sections []struct {
					Path string `json:"path"`
				} `json:"Location"`
			} `json:"Directory"`
		} `json:"MediaContainer"`
	}

	resp := new(Response)
	if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
		return nil, fmt.Errorf("failed decoding libraries response: %v: %w", err, autoscan.ErrFatal)
	}

	// process response
	libraries := make([]library, 0)
	for _, lib := range resp.MediaContainer.Libraries {
		for _, folder := range lib.Sections {
			libPath := folder.Path

			// Add trailing slash if there is none.
			if len(libPath) > 0 && libPath[len(libPath)-1] != '/' {
				libPath += "/"
			}

			libraries = append(libraries, library{
				Name: lib.Name,
				ID:   lib.ID,
				Path: libPath,
			})
		}
	}

	return libraries, nil
}

func (c apiClient) Scan(path string, libraryID int) error {
	reqURL := autoscan.JoinURL(c.baseURL, "library", "sections", strconv.Itoa(libraryID), "refresh")
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed creating scan request: %v: %w", err, autoscan.ErrFatal)
	}

	q := url.Values{}
	q.Add("path", path)
	req.URL.RawQuery = q.Encode()

	res, err := c.do(req)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	res.Body.Close()
	return nil
}
