package emby

import (
	"fmt"
	"net/http"

	"github.com/cloudbox/autoscan"
)

func (t target) Available() error {
	// create request
	req, err := http.NewRequest("GET", autoscan.JoinURL(t.url, "emby", "SystemInfo", "Info"), nil)
	if err != nil {
		return fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	// set headers
	req.Header.Set("X-Emby-Token", t.token)
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
