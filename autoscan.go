package autoscan

import (
	"net/http"
	"path"
	"strings"
)

// A Scan is at the core of Autoscan.
// It defines which path to scan and with which (trigger-given) priority.
//
// The Scan is used across Triggers, Targets and the Processor.
type Scan struct {
	Path     string
	Priority int
	Metadata Metadata
}

// Metadata is an optional extension to autoscan.Scan.
// It defines the provider (e.g. IMDb or TVDb) and the corresponding ID.
//
// Metadata MAY be used by targets to get a perfect match.
type Metadata struct {
	Provider string
	ID       string
}

// BaseConfig defines the base config all Triggers and Targets MUST support.
type BaseConfig struct {
	AddPrefix   string `yaml:"addPrefix"`
	StripPrefix string `yaml:"stripPrefix"`
}

// TriggerConfig extends the BaseConfig and adds Trigger-specific fields.
type TriggerConfig struct {
	BaseConfig `yaml:",inline"`
	Priority   int `yaml:"priority"`
}

// HTTPTriggerConfig extends the TriggerConfig and adds HTTP-related fields.
type HTTPTriggerConfig struct {
	TriggerConfig `yaml:",inline"`

	// Name MUST be used by the router to separate instances of the same trigger.
	Name string `yaml:"name"`
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
	// TVDb provider for use in autoscan.Metadata
	TVDb = "tvdb"

	// TMDb provider for use in autoscan.Metadata
	TMDb = "tmdb"

	// IMDb provider for use in autoscan.Metadata
	IMDb = "imdb"
)

// AddPrefix adds the specified prefix to the given path.
// If the prefix does not start with the slash, it gets added.
func AddPrefix(prefix string, p string) string {
	return path.Join("/", prefix, p)
}

// StripPrefix removes the specified prefix from the given path.
// A slash at the end of the prefix will be ignored.
func StripPrefix(prefix string, p string) string {
	return path.Join("/", strings.TrimPrefix(p, prefix))
}
