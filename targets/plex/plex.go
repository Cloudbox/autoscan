package plex

import (
	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Database string `yaml:"database"`
	URL      string `yaml:"url"`
	Token    string `yaml:"token"`
}

type Target struct {
	url   string
	token string
	store *Datastore
}

func New(c Config) (*Target, error) {
	store, err := NewDatastore(c.Database)
	if err != nil {
		return nil, err
	}

	return &Target{
		url:   c.URL,
		token: c.Token,
		store: store,
	}, nil
}

func (t Target) Scan(scans []autoscan.Scan) error {
	log.Info().
		Msgf("Scanning: %+v", scans)
	return nil
}
