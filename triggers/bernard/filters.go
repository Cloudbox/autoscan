package bernard

import (
	"fmt"
	"regexp"
)

type filterer func(string) bool

func newFilterer(includes []string, excludes []string) (filterer, error) {
	reIncludes := make([]regexp.Regexp, 0)
	reExcludes := make([]regexp.Regexp, 0)

	// compile patterns
	for _, pattern := range includes {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed compiling include: %v: %w", pattern, err)
		}
		reIncludes = append(reIncludes, *re)
	}

	for _, pattern := range excludes {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed compiling exclude: %v: %w", pattern, err)
		}
		reExcludes = append(reExcludes, *re)
	}

	// create filterer
	var fn filterer = func(string) bool { return true }

	switch {
	case len(includes) > 0:
		// includes
		fn = func(path string) bool {
			for _, re := range reIncludes {
				if re.MatchString(path) {
					return true
				}
			}
			return false
		}

	case len(excludes) > 0:
		// excludes
		fn = func(path string) bool {
			for _, re := range reExcludes {
				if re.MatchString(path) {
					return false
				}
			}
			return true
		}
	}

	return fn, nil
}
