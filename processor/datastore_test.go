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

func TestAddScans(t *testing.T) {
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
			store, err := newDatastore(":memory:")
			if err != nil {
				t.Fatal(err)
			}

			var currentTime time.Time
			store.now = func() time.Time {
				currentTime = currentTime.Add(1)
				return currentTime
			}

			err = store.AddScans(tc.Scans)
			if err != nil {
				t.Fatal(err)
			}

			scan, scanTime := GetScan(t, store.db, tc.Want.Scan.Folder, tc.Want.Scan.File)
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

func TestGetMatchingScans(t *testing.T) {
	type ScanWithTime struct {
		Scan autoscan.Scan
		Time time.Time
	}

	type Test struct {
		Name  string
		Now   time.Time
		Scans []ScanWithTime
		Want  []autoscan.Scan
	}

	testTime := time.Now()

	var testCases = []Test{
		{
			Name: "Retrieves no items if all items are too young",
			Now:  testTime,
			Scans: []ScanWithTime{
				{
					Time: testTime.Add(-1 * time.Minute),
					Scan: autoscan.Scan{
						File: "1",
					},
				},
				{
					Time: testTime.Add(-1 * time.Minute),
					Scan: autoscan.Scan{
						File: "2",
					},
				},
			},
		},
		{
			Name: "Retrieves no items if some items are too young",
			Now:  testTime,
			Scans: []ScanWithTime{
				{
					Time: testTime.Add(-1 * time.Minute),
					Scan: autoscan.Scan{
						File: "1",
					},
				},
				{
					Time: testTime.Add(-10 * time.Minute),
					Scan: autoscan.Scan{
						File: "2",
					},
				},
			},
		},
		{
			Name: "Retrieves all items if all items are older than 5 minutes",
			Now:  testTime,
			Scans: []ScanWithTime{
				{
					Time: testTime.Add(-6 * time.Minute),
					Scan: autoscan.Scan{
						File: "1",
					},
				},
				{
					Time: testTime.Add(-6 * time.Minute),
					Scan: autoscan.Scan{
						File: "2",
					},
				},
			},
			Want: []autoscan.Scan{
				{
					File: "1",
				},
				{
					File: "2",
				},
			},
		},
		{
			Name: "Retrieves only one folder if all items are older than 5 minutes",
			Now:  testTime,
			Scans: []ScanWithTime{
				{
					Time: testTime.Add(-6 * time.Minute),
					Scan: autoscan.Scan{
						Folder: "folder 1",
						File:   "1",
					},
				},
				{
					Time: testTime.Add(-6 * time.Minute),
					Scan: autoscan.Scan{
						Folder: "folder 2",
						File:   "1",
					},
				},
			},
			Want: []autoscan.Scan{
				{
					Folder: "folder 1",
					File:   "1",
				},
			},
		},
		{
			Name: "Returns all fields",
			Now:  testTime,
			Scans: []ScanWithTime{
				{
					Time: testTime.Add(-6 * time.Minute),
					Scan: autoscan.Scan{
						Folder:   "Amazing folder",
						File:     "Wholesome file",
						Priority: 69,
						Size:     420,
						Metadata: autoscan.Metadata{
							ID:       "tt0417299",
							Provider: autoscan.IMDb,
						},
					},
				},
			},
			Want: []autoscan.Scan{
				{
					Folder:   "Amazing folder",
					File:     "Wholesome file",
					Priority: 69,
					Size:     420,
					Metadata: autoscan.Metadata{
						ID:       "tt0417299",
						Provider: autoscan.IMDb,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			store, err := newDatastore(":memory:")
			if err != nil {
				t.Fatal(err)
			}

			tx, err := store.db.Begin()
			if err != nil {
				t.Fatal(err)
			}

			var scanTime time.Time
			store.now = func() time.Time {
				return scanTime
			}

			for _, scan := range tc.Scans {
				scanTime = scan.Time
				err = store.addScan(tx, scan.Scan)
				if err != nil {
					t.Fatal(err)
				}
			}

			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}

			store.now = func() time.Time {
				return tc.Now
			}

			scans, err := store.GetMatchingScans()
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(scans, tc.Want) {
				t.Log(scans)
				t.Errorf("Scans do not match")
			}
		})
	}
}
