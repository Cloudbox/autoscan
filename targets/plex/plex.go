package plex

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Database string `yaml:"database"`
	URL      string `yaml:"url"`
	Token    string `yaml:"token"`
}

type Target struct {
	url       string
	token     string
	store     *Datastore
	libraries []Library

	log zerolog.Logger
}

func New(c Config) (*Target, error) {
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
		Str("url", c.URL).Logger()

	lc.Debug().
		Msgf("Retrieved %d libraries: %+v", len(libraries), libraries)

	return &Target{
		url:       c.URL,
		token:     c.Token,
		store:     store,
		libraries: libraries,

		log: lc,
	}, nil
}
