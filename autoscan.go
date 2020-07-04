package autoscan

import "net/http"

// A Scan is at the core of Autoscan.
// It defines which path to scan and with which (trigger-given) priority.
//
// The Scan is used across Triggers, Targets and the Processor.
type Scan struct {
	Path     string
	Priority int
	Metadata Metadata
}

type Metadata struct {
	Provider string
	ID       string
}

// A Trigger pushes new Scans to the given channel.
//
// The Trigger should be used for background processes
// which fetch new events every x number of seconds.
//
// A Trigger MUST translate the original event into a Scan
// and push it to the channel.
type Trigger func(chan Scan)

// A HTTPTrigger is a Trigger which does not run in the background,
// and instead returns a http.Handler.
//
// This http.Handler should be added to the autoscan router in cmd/autoscan.
type HTTPTrigger func(chan Scan) http.Handler

// A Target receives a Scan from the Processor and translates the Scan
// into a format understood by the target.
type Target func(Scan)

const (
	TMDb = "tmdb"
	IMDb = "imdb"
)
