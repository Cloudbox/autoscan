package emby

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog"
	"net/http"
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

func (c apiClient) validateResponseStatus(requestType string, successStatus int, res *http.Response,
	unexpectedErrorType error) error {
	switch res.StatusCode {
	case successStatus:
		// success
		return nil
	case 401:
		// unauthorized
		return fmt.Errorf("emby token is invalid: failed validating %v request response: %w",
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
	reqURL := autoscan.JoinURL(c.url, "emby", "System", "Info")
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed creating availability request: %v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("X-Emby-Token", c.token)
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

	// set headers
	req.Header.Set("X-Emby-Token", c.token)
	req.Header.Set("Accept", "application/json")

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
	err = c.validateResponseStatus("scan", 204, res, autoscan.ErrRetryScan)
	if err != nil {
		return err
	}

	return nil
}
