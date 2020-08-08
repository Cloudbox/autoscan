package autoscan

import (
	"testing"
)

func TestRewriter(t *testing.T) {
	type Test struct {
		Name     string
		Rewrites []Rewrite
		Input    string
		Expected string
	}

	var testCases = []Test{
		{
			Name:     "One parameter with wildcard",
			Input:    "/mnt/unionfs/Media/Movies/Example Movie/movie.mkv",
			Expected: "/data/Movies/Example Movie/movie.mkv",
			Rewrites: []Rewrite{{
				From: "/mnt/unionfs/Media/*",
				To:   "/data/$1",
			}},
		},
		{
			Name:     "One parameter with glob thingy",
			Input:    "/Media/Movies/test.mkv",
			Expected: "/data/Movies/test.mkv",
			Rewrites: []Rewrite{{
				From: "/Media/(.*)",
				To:   "/data/$1",
			}},
		},
		{
			Name:     "No wildcard",
			Input:    "/Media/whatever",
			Expected: "/whatever",
			Rewrites: []Rewrite{{
				From: "^/Media/",
				To:   "/$1",
			}},
		},
		{
			Name:     "Unicode (PAS issue #73)",
			Input:    "/media/b33f/saitoh183/private/Videos/FrenchTV/L'échappée/Season 03",
			Expected: "/Videos/FrenchTV/L'échappée/Season 03",
			Rewrites: []Rewrite{{
				From: "/media/b33f/saitoh183/private/*",
				To:   "/$1",
			}},
		},
		{
			Name:     "Returns input when no rules are given",
			Input:    "/mnt/unionfs/test/example.mp4",
			Expected: "/mnt/unionfs/test/example.mp4",
		},
		{
			Name:     "Returns input when rule does not match",
			Input:    "/test/example.mp4",
			Expected: "/test/example.mp4",
			Rewrites: []Rewrite{{
				From: "^/Media/*",
				To:   "/mnt/unionfs/Media/$1",
			}},
		},
		{
			Name:     "Uses second rule if first one does not match",
			Input:    "/test/example.mp4",
			Expected: "/mnt/unionfs/example.mp4",
			Rewrites: []Rewrite{
				{From: "^/Media/*", To: "/mnt/unionfs/Media/$1"},
				{From: "^/test/*", To: "/mnt/unionfs/$1"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			rewriter, err := NewRewriter(tc.Rewrites)

			if err != nil {
				t.Fatal(err)
			}

			result := rewriter(tc.Input)
			if result != tc.Expected {
				t.Errorf("%s does not equal %s", result, tc.Expected)
			}
		})
	}

}
