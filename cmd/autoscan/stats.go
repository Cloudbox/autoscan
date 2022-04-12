package main

import (
	"errors"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudbox/autoscan"
	"github.com/cloudbox/autoscan/processor"
)

func scanStats(proc *processor.Processor, interval time.Duration) {
	st := time.NewTicker(interval)
	for {
		select {
		case _ = <-st.C:
			// retrieve amount of scans remaining
			sm, err := proc.ScansRemaining()
			switch {
			case err == nil:
				log.Info().
					Int("remaining", sm).
					Int64("processed", proc.ScansProcessed()).
					Msg("Scan stats")
			case errors.Is(err, autoscan.ErrFatal):
				log.Error().
					Err(err).
					Msg("Fatal error determining amount of remaining scans, scan stats stopped...")
				st.Stop()
				return
			default:
				// ErrNoScans should never occur as COUNT should always at-least return 0
				log.Error().
					Err(err).
					Msg("Failed determining amount of remaining scans")
			}
		}
	}
}
