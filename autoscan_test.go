package autoscan

import (
	"testing"
)

func TestRewriter(t *testing.T) {
	type Test struct {
		Name     string
		From     string
		To       string
		Input    string
		Expected string
	}

	var testCases = []Test{
		{
			Name:     "One parameter with wildcard",
			From:     "/mnt/unionfs/Media/*",
			To:       "/data/$1",
			Input:    "/mnt/unionfs/Media/Movies/Example Movie/movie.mkv",
			Expected: "/data/Movies/Example Movie/movie.mkv",
		},
		{
			Name:     "One parameter with glob thingy",
			From:     "/Media/(.*)",
			To:       "/data/$1",
			Input:    "/Media/Movies/test.mkv",
			Expected: "/data/Movies/test.mkv",
		},
		{
			Name:     "No wildcard",
			From:     "^/Media/",
			To:       "/$1",
			Input:    "/Media/whatever",
			Expected: "/whatever",
		},
		{
			Name:     "Unicode (PAS issue #73)",
			From:     "/media/b33f/saitoh183/private/*",
			To:       "/$1",
			Input:    "/media/b33f/saitoh183/private/Videos/FrenchTV/L'échappée/Season 03",
			Expected: "/Videos/FrenchTV/L'échappée/Season 03",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			rewriter, err := NewRewriter(Rewrite{
				From: tc.From,
				To:   tc.To,
			})

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
