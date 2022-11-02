package readarr

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
		Name:     "readarr",
		Priority: 5,
		Rewrite: []autoscan.Rewrite{{
			From: "/Books/*",
			To:   "/mnt/unionfs/Media/Books/$1",
		}},
	}

	currentTime := time.Now()
	now = func() time.Time {
		return currentTime
	}

	var testCases = []Test{
		{
			"Scan has all the correct fields",
			Given{
				Config:  standardConfig,
				Fixture: "testdata/sanderson.json",
			},
			Expected{
				StatusCode: 200,
				Scans: []autoscan.Scan{{
					Folder:   "/mnt/unionfs/Media/Books/Brandon Sanderson/The Way of Kings (2010)",
					Priority: 5,
					Time:     currentTime,
				}},
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
					t.Logf("want: %v", tc.Expected.Scans)
					t.Logf("got:  %v", scans)
					t.Errorf("Scans do not equal")
					return errors.New("Scans do not equal")
				}

				return nil
			}

			trigger, err := New(tc.Given.Config)
			if err != nil {
				t.Fatalf("Could not create Readarr Trigger: %v", err)
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
