package plex

import (
	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Database string           `yaml:"database"`
	URL      string           `yaml:"url"`
	Token    string           `yaml:"token"`
	Rewrite  autoscan.Rewrite `yaml:"rewrite"`
}

type target struct {
	url       string
	token     string
	libraries []Library

	log     zerolog.Logger
	rewrite autoscan.Rewriter
	store   *Datastore
}

func New(c Config) (*target, error) {
	rewriter, err := autoscan.NewRewriter(c.Rewrite)
	if err != nil {
		return nil, err
	}

	store, err := NewDatastore(c.Database)
	if err != nil {
		return nil, err
	}

	libraries, err := store.Libraries()
	if err != nil {
		return nil, err
	}

	lc := log.With().
		Str("target", "plex").
		Str("target_url", c.URL).Logger()

	lc.Debug().
		Interface("libraries", libraries).
		Msgf("Retrieved %d libraries", len(libraries))

	return &target{
		url:       c.URL,
		token:     c.Token,
		libraries: libraries,

		log:     lc,
		rewrite: rewriter,
		store:   store,
	}, nil
}
