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
	c Config
}

func (t Target) Scan(scans []autoscan.Scan) error {
	log.Info().Msgf("Scanning: %+v", scans)
	return nil
}

func New(c Config) Target {
	return Target{c: c}
}
