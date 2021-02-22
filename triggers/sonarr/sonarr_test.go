package sonarr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/cloudbox/autoscan"
)

func TestHandler(t *testing.T) {
	type Given struct {
		Config  Config
		Fixture string
	}

	type Expected struct {
		Scans      []autoscan.Scan
		StatusCode int
	}

	type Test struct {
		Name     string
		Given    Given
		Expected Expected
	}

	standardConfig := Config{
		Name:     "sonarr",
		Priority: 5,
		Rewrite: []autoscan.Rewrite{{
			From: "/TV/*",
			To:   "/mnt/unionfs/Media/TV/$1",
		}},
	}

	currentTime := time.Now()
	now = func() time.Time {
		return currentTime
	}

	var testCases = []Test{
		{
			"Scan has all the correct fields on Download event",
			Given{
				Config:  standardConfig,
				Fixture: "testdata/westworld.json",
			},
			Expected{
				StatusCode: 200,
				Scans: []autoscan.Scan{
					{
						Folder:   "/mnt/unionfs/Media/TV/Westworld/Season 1",
						Priority: 5,
						Time:     currentTime,
					},
				},
			},
		},
		{
			"Scan on EpisodeFileDelete",
			Given{
				Config:  standardConfig,
				Fixture: "testdata/episode_delete.json",
			},
			Expected{
				StatusCode: 200,
				Scans: []autoscan.Scan{
					{
						Folder:   "/mnt/unionfs/Media/TV/Westworld/Season 2",
						Priority: 5,
						Time:     currentTime,
					},
				},
			},
		},
		{
			"Picks up the Rename event without duplicates",
			Given{
				Config:  standardConfig,
				Fixture: "testdata/rename.json",
			},
			Expected{
				StatusCode: 200,
				Scans: []autoscan.Scan{
					{
						Folder:   "/mnt/unionfs/Media/TV/Westworld/Season 1",
						Priority: 5,
						Time:     currentTime,
					},
					{
						Folder:   "/mnt/unionfs/Media/TV/Westworld [imdb:tt0475784]/Season 1",
						Priority: 5,
						Time:     currentTime,
					},
					{
						Folder:   "/mnt/unionfs/Media/TV/Westworld/Season 2",
						Priority: 5,
						Time:     currentTime,
					},
					{
						Folder:   "/mnt/unionfs/Media/TV/Westworld [imdb:tt0475784]/Season 2",
						Priority: 5,
						Time:     currentTime,
					},
				},
			},
		},
		{
			"Scans show folder on SeriesDelete event",
			Given{
				Config:  standardConfig,
				Fixture: "testdata/series_delete.json",
			},
			Expected{
				StatusCode: 200,
				Scans: []autoscan.Scan{
					{
						Folder:   "/mnt/unionfs/Media/TV/Westworld",
						Priority: 5,
						Time:     currentTime,
					},
				},
			},
		},
		{
			"Returns bad request on invalid JSON",
			Given{
				Config:  standardConfig,
				Fixture: "testdata/invalid.json",
			},
			Expected{
				StatusCode: 400,
			},
		},
		{
			"Returns 200 on Test event without emitting a scan",
			Given{
				Config:  standardConfig,
				Fixture: "testdata/test.json",
			},
			Expected{
				StatusCode: 200,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			callback := func(scans ...autoscan.Scan) error {
				if !reflect.DeepEqual(tc.Expected.Scans, scans) {
					t.Log(scans)
					t.Log(tc.Expected.Scans)
					t.Errorf("Scans do not equal")
					return errors.New("Scans do not equal")
				}

				return nil
			}

			trigger, err := New(tc.Given.Config)
			if err != nil {
				t.Fatalf("Could not create Sonarr Trigger: %v", err)
			}

			server := httptest.NewServer(trigger(callback))
			defer server.Close()

			request, err := os.Open(tc.Given.Fixture)
			if err != nil {
				t.Fatalf("Could not open the fixture: %s", tc.Given.Fixture)
			}

			res, err := http.Post(server.URL, "application/json", request)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			defer res.Body.Close()
			if res.StatusCode != tc.Expected.StatusCode {
				t.Errorf("Status codes do not match: %d vs %d", res.StatusCode, tc.Expected.StatusCode)
			}
		})
	}
}
