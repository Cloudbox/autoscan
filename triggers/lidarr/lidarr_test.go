package lidarr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/cloudbox/autoscan"
)

func TestHandler(t *testing.T) {
	type Given struct {
		Config  Config
		Fixture string
		Sizes   map[string]uint64
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
		Name:     "lidarr",
		Priority: 5,
		Rewrite: autoscan.Rewrite{
			From: "/Music/*",
			To:   "/mnt/unionfs/Media/Music/$1",
		},
	}

	var testCases = []Test{
		{
			"Scan has all the correct fields",
			Given{
				Config:  standardConfig,
				Fixture: "testdata/marshmello.json",
				Sizes: map[string]uint64{
					"01 - Down.mp3":            100,
					"02 - Run It Up.mp3":       500,
					"03 - Put Yo Hands Up.mp3": 200,
					"04 - Let’s Get Down.mp3":  1200,
				},
			},
			Expected{
				StatusCode: 200,
				Scans: []autoscan.Scan{
					{
						File:     "01 - Down.mp3",
						Folder:   "/mnt/unionfs/Media/Music/Marshmello/Joytime III (2019)",
						Priority: 5,
						Size:     100,
					},
					{
						File:     "02 - Run It Up.mp3",
						Folder:   "/mnt/unionfs/Media/Music/Marshmello/Joytime III (2019)",
						Priority: 5,
						Size:     500,
					},
					{
						File:     "03 - Put Yo Hands Up.mp3",
						Folder:   "/mnt/unionfs/Media/Music/Marshmello/Joytime III (2019)",
						Priority: 5,
						Size:     200,
					},
					{
						File:     "04 - Let’s Get Down.mp3",
						Folder:   "/mnt/unionfs/Media/Music/Marshmello/Joytime III (2019)",
						Priority: 5,
						Size:     1200,
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
			fileSize = func(name string) (uint64, error) {
				return tc.Given.Sizes[filepath.Base(name)], nil
			}

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
				t.Fatalf("Could not create Lidarr Trigger: %v", err)
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
