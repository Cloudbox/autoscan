package radarr

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/cloudbox/autoscan"
)

func callHandler(t *testing.T, config Config, fixture string) (*http.Response, *autoscan.Scan) {
	t.Helper()

	trigger, err := New(config)
	if err != nil {
		t.Fatalf("Could not create Radarr Trigger: %v", err)
	}

	scans := make(chan autoscan.Scan, 1)
	server := httptest.NewServer(trigger(scans))
	defer server.Close()

	request, err := os.Open(fixture)
	if err != nil {
		t.Fatalf("Could not open the fixture: %s", fixture)
	}

	res, err := http.Post(server.URL, "application/json", request)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	select {
	case scan := <-scans:
		return res, &scan
	default:
		return res, nil
	}
}

func TestHandler(t *testing.T) {
	type Given struct {
		Config  Config
		Fixture string
		Size    int64
	}

	type Expected struct {
		Scan       *autoscan.Scan
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
				Scan: &autoscan.Scan{
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
				Scan: &autoscan.Scan{
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

			res, scan := callHandler(t, tc.Given.Config, tc.Given.Fixture)
			defer res.Body.Close()

			if res.StatusCode != tc.Expected.StatusCode {
				t.Errorf("Status codes do not match: %d vs %d", res.StatusCode, tc.Expected.StatusCode)
			}

			if !reflect.DeepEqual(tc.Expected.Scan, scan) {
				t.Log(scan)
				t.Errorf("Scans do not equal")
			}
		})
	}
}
