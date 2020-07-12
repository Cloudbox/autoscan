package processor

import (
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/cloudbox/autoscan"

	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

const sqlGetScan = `
SELECT folder, file, priority, size, time, meta_provider, meta_id FROM scan
WHERE folder = $1 AND file = $2
`

func GetScan(t *testing.T, db *sql.DB, folder string, file string) (scan autoscan.Scan, scanTime time.Time) {
	t.Helper()

	var metaProvider sql.NullString
	var metaID sql.NullString

	row := db.QueryRow(sqlGetScan, folder, file)

	err := row.Scan(&scan.Folder, &scan.File, &scan.Priority, &scan.Size, &scanTime, &metaProvider, &metaID)
	if err != nil {
		t.Fatalf("Could not scan the row: %v", err)
	}

	if metaID.String == "" && metaID.Valid {
		t.Fatal("Meta ID is an empty string, not a NULL")
	}

	if metaProvider.String == "" && metaProvider.Valid {
		t.Fatal("Meta Provider is an empty string, not a NULL")
	}

	scan.Metadata = autoscan.Metadata{
		ID:       metaID.String,
		Provider: metaProvider.String,
	}

	return
}

func TestProcessTriggers(t *testing.T) {
	type Want struct {
		Scan autoscan.Scan
		Time time.Time
	}

	type Test struct {
		Name  string
		Scans []autoscan.Scan
		Want  Want
	}

	var testCases = []Test{
		{
			Name: "Empty Metadata fields should return NULL",
			Scans: []autoscan.Scan{
				{
					Folder:   "testfolder/test",
					File:     "test.mkv",
					Priority: 5,
					Size:     300,
				},
			},
			Want: Want{
				Time: time.Time{}.Add(1),
				Scan: autoscan.Scan{
					Folder:   "testfolder/test",
					File:     "test.mkv",
					Priority: 5,
					Size:     300,
				},
			},
		},
		{
			Name: "Metadata can be updated but cannot be removed",
			Scans: []autoscan.Scan{
				{
					Metadata: autoscan.Metadata{
						ID:       "",
						Provider: "",
					},
				},
				{
					Metadata: autoscan.Metadata{
						ID:       "tt1234",
						Provider: autoscan.IMDb,
					},
				},
				{
					Metadata: autoscan.Metadata{
						ID:       "tt5678",
						Provider: autoscan.IMDb,
					},
				},
				{
					Metadata: autoscan.Metadata{
						ID:       "",
						Provider: "",
					},
				},
			},
			Want: Want{
				Time: time.Time{}.Add(4),
				Scan: autoscan.Scan{
					Metadata: autoscan.Metadata{
						ID:       "tt5678",
						Provider: autoscan.IMDb,
					},
				},
			},
		},
		{
			Name: "Priority shall increase but not decrease",
			Scans: []autoscan.Scan{
				{
					Priority: 2,
				},
				{
					Priority: 5,
				},
				{
					Priority: 3,
				},
			},
			Want: Want{
				Time: time.Time{}.Add(3),
				Scan: autoscan.Scan{
					Priority: 5,
				},
			},
		},
		{
			Name: "Size should always reflect the newest value",
			Scans: []autoscan.Scan{
				{
					Size: 1,
				},
				{
					Size: 3,
				},
				{
					Size: 2,
				},
			},
			Want: Want{
				Time: time.Time{}.Add(3),
				Scan: autoscan.Scan{
					Size: 2,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			proc, err := New(":memory:")
			if err != nil {
				t.Fatal(err)
			}

			var currentTime time.Time
			proc.store.now = func() time.Time {
				return currentTime
			}

			for _, scan := range tc.Scans {
				currentTime = currentTime.Add(1)

				err = proc.AddScan(scan)
				if err != nil {
					t.Fatal(err)
				}
			}

			scan, scanTime := GetScan(t, proc.store.db, tc.Want.Scan.Folder, tc.Want.Scan.File)
			if !reflect.DeepEqual(tc.Want.Scan, scan) {
				t.Log(scan)
				t.Errorf("Scans do not equal")
			}

			if scanTime != tc.Want.Time {
				t.Log(scanTime)
				t.Errorf("Scan times do not equal")
			}
		})
	}
}
