package plex

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cloudbox/autoscan"
)

type apiClient struct {
	client *http.Client
	url    string
	token  string
}

func newAPIClient(c Config) *apiClient {
	return &apiClient{
		client: &http.Client{},
		url:    c.URL,
		token:  c.Token,
	}
}

func (c apiClient) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("X-Plex-Token", c.token)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, autoscan.ErrTargetUnavailable)
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, nil
	}

	// statusCode not in the 2xx range, close response
	res.Body.Close()

	switch res.StatusCode {
	case 401:
		return nil, fmt.Errorf("invalid plex token: %s: %w", res.Status, autoscan.ErrFatal)
	case 404, 500, 503, 504:
		return nil, fmt.Errorf("%s: %w", res.Status, autoscan.ErrTargetUnavailable)
	default:
		return nil, fmt.Errorf("%s: %w", res.Status, autoscan.ErrFatal)
	}
}

func (c apiClient) Available() error {
	// create request
	reqURL := autoscan.JoinURL(c.url, "myplex", "account")
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed creating availability request: %v: %w", err, autoscan.ErrFatal)
	}

	// send request
	res, err := c.do(req)
	if err != nil {
		return fmt.Errorf("availability: %w", err)
	}

	res.Body.Close()
	return nil
}

type library struct {
	ID   int
	Name string
	Path string
}

func (c apiClient) Libraries() ([]library, error) {
	// create request
	reqURL := autoscan.JoinURL(c.url, "library", "sections")
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
		Library []struct {
			Name    string `xml:"title,attr"`
			ID      int    `xml:"key,attr"`
			Section []struct {
				Path string `xml:"path,attr"`
			} `xml:"Location"`
		} `xml:"Directory"`
	}

	resp := new(Response)
	if err := xml.NewDecoder(res.Body).Decode(resp); err != nil {
		return nil, fmt.Errorf("failed decoding libraries request response: %v: %w", err, autoscan.ErrFatal)
	}

	// process response
	libraries := make([]library, 0)
	for _, lib := range resp.Library {
		for _, folder := range lib.Section {
			libraries = append(libraries, library{
				Name: lib.Name,
				ID:   lib.ID,
				Path: folder.Path,
			})
		}
	}

	return libraries, nil
}

func (c apiClient) Scan(path string, libraryID int) error {
	// create request
	reqURL := autoscan.JoinURL(c.url, "library", "sections", strconv.Itoa(libraryID), "refresh")
	req, err := http.NewRequest("PUT", reqURL, nil)
	if err != nil {
		// May only occur when the user has provided an invalid URL
		return fmt.Errorf("failed creating scan request: %v: %w", err, autoscan.ErrFatal)
	}

	// set params
	q := url.Values{}
	q.Add("path", path)
	req.URL.RawQuery = q.Encode()

	// send request
	res, err := c.do(req)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	res.Body.Close()
	return nil
}
