package autoscan

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func GetLogger(verbosity string) zerolog.Logger {
	if verbosity == "" {
		return log.Logger
	}

	level, err := zerolog.ParseLevel(verbosity)
	if err != nil {
		return log.Logger
	}

	return log.Level(level)
}
