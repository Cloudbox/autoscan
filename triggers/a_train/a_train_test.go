package a_train

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cloudbox/autoscan"
)

func TestHandler(t *testing.T) {
	type Given struct {
		Config  Config
		ID      string
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
		Drives: []Drive{{
			ID: "1234VA",
			Rewrite: []autoscan.Rewrite{{
				From: "^/TV/*",
				To:   "/mnt/unionfs/Media/TV/$1",
			}},
		}},
		Priority: 5,
		Rewrite: []autoscan.Rewrite{{
			From: "^/Movies/*",
			To:   "/mnt/unionfs/Media/Movies/$1",
		}},
	}

	currentTime := time.Now()
	now = func() time.Time {
		return currentTime
	}

	var testCases = []Test{
		{
			"Multiple created and deleted",
			Given{
				Config:  standardConfig,
				ID:      "1234VA",
				Fixture: "testdata/full.json",
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
						Folder:   "/mnt/unionfs/Media/TV/Legion/Season 1",
						Priority: 5,
						Time:     currentTime,
					},
					{
						Folder:   "/mnt/unionfs/Media/Movies/Wonder Woman 1984 (2020)",
						Priority: 5,
						Time:     currentTime,
					},
					{
						Folder:   "/mnt/unionfs/Media/Movies/Mortal Kombat (2021)",
						Priority: 5,
						Time:     currentTime,
					},
				},
			},
		},
		{
			"Duplicate (parent is given in both create and delete)",
			Given{
				Config:  standardConfig,
				ID:      "anotherVA",
				Fixture: "testdata/modified.json",
			},
			Expected{
				StatusCode: 200,
				Scans: []autoscan.Scan{
					{
						Folder:   "/TV/Legion/Season 1",
						Priority: 5,
						Time:     currentTime,
					},
					{
						Folder:   "/TV/Legion/Season 1",
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
					t.Log(scans)
					t.Log(tc.Expected.Scans)
					t.Errorf("Scans do not equal")
					return errors.New("Scans do not equal")
				}

				return nil
			}

			trigger, err := New(tc.Given.Config)
			if err != nil {
				t.Fatalf("Could not create A-Train Trigger: %v", err)
			}

			r := chi.NewRouter()
			r.Post("/triggers/a-train/{drive}", trigger(callback).ServeHTTP)

			server := httptest.NewServer(r)
			defer server.Close()

			request, err := os.Open(tc.Given.Fixture)
			if err != nil {
				t.Fatalf("Could not open the fixture: %s", tc.Given.Fixture)
			}

			url := fmt.Sprintf("%s/triggers/a-train/%s", server.URL, tc.Given.ID)
			res, err := http.Post(url, "application/json", request)
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
