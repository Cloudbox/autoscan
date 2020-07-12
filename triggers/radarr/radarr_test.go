package radarr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/cloudbox/autoscan"
)

func TestHandler(t *testing.T) {
	type Given struct {
		Config  Config
		Fixture string
		Size    int64
	}

	type Expected struct {
		Scan       autoscan.Scan
		StatusCode int
	}

	type Test struct {
		Name     string
		Given    Given
		Expected Expected
	}

	standardConfig := Config{
		Name:     "radarr",
		Priority: 5,
		Rewrite: autoscan.Rewrite{
			From: "/Movies/*",
			To:   "/mnt/unionfs/Media/Movies/$1",
		},
	}

	var testCases = []Test{
		{
			"Returns IMDb if both TMDb and IMDb are given",
			Given{
				Config:  standardConfig,
				Fixture: "testdata/interstellar.json",
				Size:    157336,
			},
			Expected{
				StatusCode: 200,
				Scan: autoscan.Scan{
					File:   "Interstellar.2014.UHD.BluRay.2160p.REMUX.mkv",
					Folder: "/mnt/unionfs/Media/Movies/Interstellar (2014)",
					Metadata: autoscan.Metadata{
						ID:       "tt0816692",
						Provider: autoscan.IMDb,
					},
					Priority: 5,
					Size:     157336,
				},
			},
		},
		{
			"Returns TMDb if no IMDb is given",
			Given{
				Config: Config{
					Name:     "radarr",
					Priority: 3,
					Rewrite: autoscan.Rewrite{
						From: "/data/*",
						To:   "/Media/$1",
					},
				},
				Fixture: "testdata/parasite.json",
				Size:    200000,
			},
			Expected{
				StatusCode: 200,
				Scan: autoscan.Scan{
					File:   "Parasite.2019.2160p.UHD.BluRay.REMUX.HEVC.TrueHD.Atmos.7.1.mkv",
					Folder: "/Media/Movies/Parasite (2019)",
					Metadata: autoscan.Metadata{
						ID:       "496243",
						Provider: autoscan.TMDb,
					},
					Priority: 3,
					Size:     200000,
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
			fileSize = func(name string) (int64, error) {
				return tc.Given.Size, nil
			}

			callback := func(scan autoscan.Scan) error {
				if !reflect.DeepEqual(tc.Expected.Scan, scan) {
					t.Log(scan)
					t.Log(tc.Expected.Scan)
					t.Errorf("Scans do not equal")
					return errors.New("Scans do not equal")
				}

				return nil
			}

			trigger, err := New(tc.Given.Config)
			if err != nil {
				t.Fatalf("Could not create Radarr Trigger: %v", err)
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
