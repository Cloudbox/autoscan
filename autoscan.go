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
	Priority int
	Retries  int
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

	// ErrFatal indicates a severe problem related to development.
	ErrFatal = errors.New("fatal error")

	// ErrNoScans is not an error. It only indicates whether the CLI
	// should sleep longer depending on the processor output.
	ErrNoScans = errors.New("no scans currently available")

	// ErrAnchorUnavailable indicates that an Anchor file is
	// not available on the file system. Processing should halt
	// until all anchors are available.
	ErrAnchorUnavailable = errors.New("anchor file is unavailable")
)

type Rewrite struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type Rewriter func(string) string

func NewRewriter(rewriteRules []Rewrite) (Rewriter, error) {
	var rewrites []regexp.Regexp
	for _, rule := range rewriteRules {
		re, err := regexp.Compile(rule.From)
		if err != nil {
			return nil, err
		}

		rewrites = append(rewrites, *re)
	}

	rewriter := func(input string) string {
		for i, r := range rewrites {
			if r.MatchString(input) {
				return r.ReplaceAllString(input, rewriteRules[i].To)
			}
		}

		return input
	}

	return rewriter, nil
}
