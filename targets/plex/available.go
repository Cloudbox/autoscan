package plex

import (
	"net/http"

	"github.com/cloudbox/autoscan"
)

func (t target) Available() bool {
	// create request
	req, err := http.NewRequest("GET", autoscan.JoinURL(t.url, "myplex", "account"), nil)
	if err != nil {
		t.log.Error().
			Err(err).
			Msg("Failed creating availability check request")
		return false
	}

	// set headers
	req.Header.Set("X-Plex-Token", t.token)
	req.Header.Set("Accept", "application/json")

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.log.Error().
			Err(err).
			Msg("Failed sending availability check request")
		return false
	}

	defer res.Body.Close()

	// validate response
	if res.StatusCode != 200 {
		t.log.Error().
			Str("status", res.Status).
			Msg("Failed validating availability check response")
		return false
	}

	return true
}
