package autoscan

import (
	"errors"
	"net/http"
	"regexp"
)

// A Scan is at the core of Autoscan.
// It defines which path to scan and with which (trigger-given) priority.
//
// The Scan is used across Triggers, Targets and the Processor.
type Scan struct {
	Folder   string
	File     string
	Size     uint64
	Priority int
	Retries  int
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

type ProcessorFunc func(...Scan) error

type Trigger func(ProcessorFunc)

// A HTTPTrigger is a Trigger which does not run in the background,
// and instead returns a http.Handler.
//
// This http.Handler should be added to the autoscan router in cmd/autoscan.
type HTTPTrigger func(ProcessorFunc) http.Handler

// A Target receives a Scan from the Processor and translates the Scan
// into a format understood by the target.
type Target interface {
	Scan([]Scan) error
	Available() error
}

const (
	// TVDb provider for use in autoscan.Metadata
	TVDb = "tvdb"

	// TMDb provider for use in autoscan.Metadata
	TMDb = "tmdb"

	// IMDb provider for use in autoscan.Metadata
	IMDb = "imdb"
)

var (
	// ErrTargetUnavailable may occur when a Target goes offline
	// or suffers from fatal errors. In this case, the processor
	// will halt operations until the target is back online.
	ErrTargetUnavailable = errors.New("target unavailable")

	// ErrRetryScan indicates a temporary scan-specific error.
	ErrRetryScan = errors.New("scan should be retried later")

	// ErrFatal indicates a severe problem related to development.
	ErrFatal = errors.New("fatal development related error")

	// ErrNoScans is not an error. It only indicates whether the CLI
	// should sleep longer depending on the processor output.
	ErrNoScans = errors.New("no scans currently available")
)

type Rewrite struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type Rewriter func(string) string

func NewRewriter(r Rewrite) (Rewriter, error) {
	if r.From == "" || r.To == "" {
		rewriter := func(input string) string {
			return input
		}

		return rewriter, nil
	}

	re, err := regexp.Compile(r.From)
	if err != nil {
		return nil, err
	}

	rewriter := func(input string) string {
		return re.ReplaceAllString(input, r.To)
	}

	return rewriter, nil
}
