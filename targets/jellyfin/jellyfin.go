package jellyfin

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"github.com/cloudbox/autoscan"
)

type Config struct {
	URL       string             `yaml:"url"`
	Token     string             `yaml:"token"`
	Rewrite   []autoscan.Rewrite `yaml:"rewrite"`
	Verbosity string             `yaml:"verbosity"`
}

type target struct {
	url       string
	token     string
	libraries []library

	log     zerolog.Logger
	rewrite autoscan.Rewriter
	api     apiClient
}

func New(c Config) (autoscan.Target, error) {
	l := autoscan.GetLogger(c.Verbosity).With().
		Str("target", "jellyfin").
		Str("url", c.URL).
		Logger()

	rewriter, err := autoscan.NewRewriter(c.Rewrite)
	if err != nil {
		return nil, err
	}

	api := newAPIClient(c.URL, c.Token, l)

	libraries, err := api.Libraries()
	if err != nil {
		return nil, err
	}

	l.Debug().
		Interface("libraries", libraries).
		Msg("Retrieved libraries")

	return &target{
		url:       c.URL,
		token:     c.Token,
		libraries: libraries,

		log:     l,
		rewrite: rewriter,
		api:     api,
	}, nil
}

func (t target) Available() error {
	return t.api.Available()
}

func (t target) Scan(scan autoscan.Scan) error {
	// determine library for this scan
	scanFolder := t.rewrite(scan.Folder)

	lib, err := t.getScanLibrary(scanFolder)
	if err != nil {
		t.log.Warn().
			Err(err).
			Msg("No target libraries found")

		return nil
	}

	l := t.log.With().
		Str("path", scanFolder).
		Str("library", lib.Name).
		Logger()

	// send scan request
	l.Trace().Msg("Sending scan request")

	if err := t.api.Scan(scanFolder); err != nil {
		return err
	}

	l.Info().Msg("Scan moved to target")
	return nil
}

func (t target) getScanLibrary(folder string) (*library, error) {
	for _, l := range t.libraries {
		if strings.HasPrefix(folder, l.Path) {
			return &l, nil
		}
	}

	return nil, fmt.Errorf("%v: failed determining library", folder)
}
