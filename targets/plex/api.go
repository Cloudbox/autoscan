package plex

import (
	"encoding/xml"
	"fmt"
	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog"
	"net/http"
	"net/url"
	"strconv"
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
			Str("target", "plex").
			Str("url", c.URL).
			Logger(),
	}
}

func (c apiClient) validateResponseStatus(requestType string, successStatus int, res *http.Response,
	unexpectedErrorType error) error {
	switch res.StatusCode {
	case successStatus:
		// success
		return nil
	case 401:
		// unauthorized
		return fmt.Errorf("plex token is invalid: failed validating %v request response: %w",
			requestType, autoscan.ErrFatal)
	case 503, 504:
		// unavailable
		return fmt.Errorf("%v: failed validating %v request response: %w",
			res.Status, requestType, autoscan.ErrTargetUnavailable)
	}

	return fmt.Errorf("%v: failed validating %v request response: %w",
		res.Status, requestType, unexpectedErrorType)
}

func (c apiClient) Available() error {
	// create request
	reqURL := autoscan.JoinURL(c.url, "myplex", "account")
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed creating availability request: %v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("X-Plex-Token", c.token)
	req.Header.Set("Accept", "application/json")

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed sending availability request: %v: %w", err, autoscan.ErrTargetUnavailable)
	}

	defer res.Body.Close()

	// validate response
	err = c.validateResponseStatus("availability", 200, res, autoscan.ErrTargetUnavailable)
	if err != nil {
		return err
	}

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

	// set headers
	req.Header.Set("X-Plex-Token", c.token)
	req.Header.Set("Accept", "application/xml")

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed sending libraries request: %v: %w", err, autoscan.ErrFatal)
	}

	defer res.Body.Close()

	// validate response
	err = c.validateResponseStatus("libraries", 200, res, autoscan.ErrFatal)
	if err != nil {
		return nil, err
	}

	// decode response
	type Response struct {
		Library []struct {
			Name    string `xml:"title,attr"`
			Section []struct {
				Id   int    `xml:"id,attr"`
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
				ID:   folder.Id,
				Name: lib.Name,
				Path: folder.Path,
			})
		}
	}

	return libraries, nil
}

func (c apiClient) Scan(path string, libraryId int) error {
	// create request
	reqURL := autoscan.JoinURL(c.url, "library", "sections", strconv.Itoa(libraryId), "refresh")
	req, err := http.NewRequest("PUT", reqURL, nil)
	if err != nil {
		// May only occur when the user has provided an invalid URL
		return fmt.Errorf("failed creating scan request: %v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("X-Plex-Token", c.token)

	// set params
	q := url.Values{}
	q.Add("path", path)

	req.URL.RawQuery = q.Encode()

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed sending scan request: %v: %w", err, autoscan.ErrTargetUnavailable)
	}

	defer res.Body.Close()

	// validate response
	err = c.validateResponseStatus("scan", 200, res, autoscan.ErrRetryScan)
	if err != nil {
		return err
	}

	return nil

}
