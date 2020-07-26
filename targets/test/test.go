package test

import (
	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog/log"
)

type target struct {
	count int
}

func (t *target) Scan(scans []autoscan.Scan) error {
	log.Debug().
		Interface("scans", scans).
		Msg("Received scans")

	return nil
}

func (t *target) Available() error {
	if t.count < 2 {
		t.count++
		return autoscan.ErrTargetUnavailable
	}

	return nil
}

func New() (*target, error) {
	return &target{}, nil
}
