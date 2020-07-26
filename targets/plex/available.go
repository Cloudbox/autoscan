package plex

import (
	"fmt"
	"net/http"

	"github.com/cloudbox/autoscan"
)

func (t target) Available() error {
	// create request
	req, err := http.NewRequest("GET", autoscan.JoinURL(t.url, "myplex", "account"), nil)
	if err != nil {
		return fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("X-Plex-Token", t.token)
	req.Header.Set("Accept", "application/json")

	// send request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not check Plex availability: %v: %w",
			err, autoscan.ErrTargetUnavailable)
	}

	defer res.Body.Close()

	// validate response
	if res.StatusCode != 200 {
		return fmt.Errorf("could not check Plex availability: %v: %w",
			res.StatusCode, autoscan.ErrTargetUnavailable)
	}

	return nil
}
