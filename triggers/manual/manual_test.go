package manual

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/cloudbox/autoscan"
)

func TestHandler(t *testing.T) {
	type Given struct {
		Config Config
		Query  url.Values
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
		Priority: 5,
		Rewrite: []autoscan.Rewrite{{
			From: "/Movies/*",
			To:   "/mnt/unionfs/Media/Movies/$1",
		}},
	}

	currentTime := time.Now()
	now = func() time.Time {
		return currentTime
	}

	var testCases = []Test{
		{
			"Returns 200 when given multiple directories",
			Given{
				Config: standardConfig,
				Query: url.Values{
					"dir": []string{
						"/Movies/Interstellar (2014)",
						"/Movies/Parasite (2019)",
					},
				},
			},
			Expected{
				StatusCode: 200,
				Scans: []autoscan.Scan{
					{
						Folder:   "/mnt/unionfs/Media/Movies/Interstellar (2014)",
						Priority: 5,
						Time:     currentTime,
					},
					{
						Folder:   "/mnt/unionfs/Media/Movies/Parasite (2019)",
						Priority: 5,
						Time:     currentTime,
					},
				},
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
				t.Fatalf("Could not create Manual Trigger: %v", err)
			}

			server := httptest.NewServer(trigger(callback))
			defer server.Close()

			req, err := http.NewRequest("POST", server.URL, nil)
			if err != nil {
				t.Fatalf("Failed creating request: %v", err)
			}

			req.URL.RawQuery = tc.Given.Query.Encode()

			res, err := http.DefaultClient.Do(req)
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
