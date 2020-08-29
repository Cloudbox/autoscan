package autoscan

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

// A Scan is at the core of Autoscan.
// It defines which path to scan and with which (trigger-given) priority.
//
// The Scan is used across Triggers, Targets and the Processor.
type Scan struct {
	Folder   string
	Priority int
	Time     time.Time
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
	Scan(Scan) error
	Available() error
}

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

type Filterer func(string) bool

func NewFilterer(includes []string, excludes []string) (Filterer, error) {
	reIncludes := make([]regexp.Regexp, 0)
	reExcludes := make([]regexp.Regexp, 0)

	// compile patterns
	for _, pattern := range includes {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling include: %v: %w", pattern, err)
		}
		reIncludes = append(reIncludes, *re)
	}

	for _, pattern := range excludes {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling exclude: %v: %w", pattern, err)
		}
		reExcludes = append(reExcludes, *re)
	}

	incSize := len(reIncludes)
	excSize := len(reExcludes)

	// create filterer
	var fn Filterer = func(string) bool { return true }

	if incSize > 0 || excSize > 0 {
		fn = func(path string) bool {
			// check excludes
			for _, re := range reExcludes {
				if re.MatchString(path) {
					return false
				}
			}

			// no includes (but excludes did not match)
			if incSize == 0 {
				return true
			}

			// check includes
			for _, re := range reIncludes {
				if re.MatchString(path) {
					return true
				}
			}

			// no includes passed
			return false
		}
	}

	return fn, nil
}
