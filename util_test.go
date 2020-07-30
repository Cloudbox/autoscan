package autoscan

import (
	"net/url"
	"testing"
)

func TestDSN(t *testing.T) {
	type Test struct {
		Name  string
		Path  string
		Query url.Values
		Want  string
	}

	var testCases = []Test{
		{
			Name: "readonly",
			Path: "library.db",
			Query: url.Values{
				"mode": []string{"ro"},
			},
			Want: "file://library.db?mode=ro",
		},
		{
			Name: "No Query",
			Path: "library.db",
			Want: "file://library.db",
		},
		{
			Name: "Path from root",
			Path: "/mnt/unionfs/library.db",
			Query: url.Values{
				"mode": []string{"rw"},
			},
			Want: "file:///mnt/unionfs/library.db?mode=rw",
		},
		{
			Name: "Query sorted by key",
			Path: "library.db",
			Query: url.Values{
				"cache": []string{"shared"},
				"mode":  []string{"rw"},
			},
			Want: "file://library.db?cache=shared&mode=rw",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			dsn := DSN(tc.Path, tc.Query)
			if dsn != tc.Want {
				t.Log(dsn)
				t.Log(tc.Want)
				t.Error("DSNs do not match")
			}
		})
	}
}
