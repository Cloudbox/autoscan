package processor

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/cloudbox/autoscan"

	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

type ScanWithTime struct {
	Scan autoscan.Scan
	Time time.Time
}

const sqlGetAllWithTime = `
SELECT folder, file, priority, retries, removed, time FROM scan
`

func (store datastore) GetAllWithTime() (scans []ScanWithTime, err error) {
	rows, err := store.db.Query(sqlGetAllWithTime)
	if errors.Is(err, sql.ErrNoRows) {
		return scans, nil
	}

	if err != nil {
		return scans, err
	}

	defer rows.Close()
	for rows.Next() {
		withTime := ScanWithTime{}
		scan := &withTime.Scan

		err = rows.Scan(&scan.Folder, &scan.File, &scan.Priority, &scan.Retries, &scan.Removed, &withTime.Time)
		if err != nil {
			return scans, err
		}

		withTime.Time = withTime.Time.UTC()
		scans = append(scans, withTime)
	}

	return scans, rows.Err()
}

const sqlGetScan = `
SELECT folder, file, priority, time, retries, removed FROM scan
WHERE folder = $1 AND file = $2
`

func GetScan(t *testing.T, db *sql.DB, folder string, file string) (scan autoscan.Scan, scanTime time.Time) {
	t.Helper()

	row := db.QueryRow(sqlGetScan, folder, file)

	err := row.Scan(&scan.Folder, &scan.File, &scan.Priority, &scanTime, &scan.Retries, &scan.Removed)
	if err != nil {
		t.Fatalf("Could not scan the row: %v", err)
	}

	return scan, scanTime
}

func TestUpsert(t *testing.T) {
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
			Name: "All fields",
			Scans: []autoscan.Scan{
				{
					Folder:   "testfolder/test",
					File:     "test.mkv",
					Priority: 5,
					Retries:  2,
					Removed:  true,
				},
			},
			Want: Want{
				Time: time.Time{}.Add(1),
				Scan: autoscan.Scan{
					Folder:   "testfolder/test",
					File:     "test.mkv",
					Priority: 5,
					Retries:  2,
					Removed:  true,
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
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			store, err := newDatastore(":memory:")
			if err != nil {
				t.Fatal(err)
			}

			var currentTime time.Time
			now = func() time.Time {
				currentTime = currentTime.Add(1)
				return currentTime
			}

			err = store.Upsert(tc.Scans)
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

func TestGetMatching(t *testing.T) {
	type Test struct {
		Name   string
		Now    time.Time
		MinAge time.Duration
		Scans  []ScanWithTime
		Want   []autoscan.Scan
	}

	testTime := time.Now()

	var testCases = []Test{
		{
			Name:   "Retrieves no items if all items are too young",
			Now:    testTime,
			MinAge: 2 * time.Minute,
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
			Name:   "Retrieves no items if some items are too young",
			Now:    testTime,
			MinAge: 9 * time.Minute,
			Scans: []ScanWithTime{
				{autoscan.Scan{File: "1"}, testTime.Add(-8 * time.Minute)},
				{autoscan.Scan{File: "2"}, testTime.Add(-10 * time.Minute)},
			},
		},
		{
			Name:   "Retrieves all items if all items are older than minimum age minutes",
			Now:    testTime,
			MinAge: 5 * time.Minute,
			Scans: []ScanWithTime{
				{autoscan.Scan{File: "1"}, testTime.Add(-6 * time.Minute)},
				{autoscan.Scan{File: "2"}, testTime.Add(-6 * time.Minute)},
			},
			Want: []autoscan.Scan{
				{File: "1"},
				{File: "2"},
			},
		},
		{
			Name:   "Retrieves only one folder if all items are older than minimum age minutes",
			Now:    testTime,
			MinAge: 5 * time.Minute,
			Scans: []ScanWithTime{
				{autoscan.Scan{Folder: "folder 1", File: "1"}, testTime.Add(-6 * time.Minute)},
				{autoscan.Scan{Folder: "folder 2", File: "1"}, testTime.Add(-6 * time.Minute)},
			},
			Want: []autoscan.Scan{
				{Folder: "folder 1", File: "1"},
			},
		},
		{
			Name:   "Returns all fields",
			Now:    testTime,
			MinAge: 5 * time.Minute,
			Scans: []ScanWithTime{
				{
					Time: testTime.Add(-6 * time.Minute),
					Scan: autoscan.Scan{
						Folder:   "Amazing folder",
						File:     "Wholesome file",
						Priority: 69,
						Removed:  true,
					},
				},
			},
			Want: []autoscan.Scan{
				{
					Folder:   "Amazing folder",
					File:     "Wholesome file",
					Priority: 69,
					Removed:  true,
				},
			},
		},
		{
			Name: "No scans should return an empty slice",
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
			now = func() time.Time {
				return scanTime
			}

			for _, scan := range tc.Scans {
				scanTime = scan.Time
				err = store.upsert(tx, scan.Scan)
				if err != nil {
					t.Fatal(err)
				}
			}

			err = tx.Commit()
			if err != nil {
				t.Fatal(err)
			}

			now = func() time.Time {
				return tc.Now
			}

			scans, err := store.GetMatching(tc.MinAge)
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

func TestRetries(t *testing.T) {
	type Test struct {
		Name    string
		Err     error
		Folder  string
		Retries int
		Scans   []autoscan.Scan
		Want    []ScanWithTime
	}

	testTime := time.Now().UTC()

	var testCases = []Test{
		{
			Name: "Should not error when no rows are affected",
			Err:  nil,
		},
		{
			Name:    "Only children of the same folder are incremented",
			Folder:  "1",
			Retries: 5,
			Scans: []autoscan.Scan{
				{Folder: "1", File: "1", Retries: 0},
				{Folder: "1", File: "2", Retries: 2},
				{Folder: "2", File: "1", Retries: 0},
			},
			Want: []ScanWithTime{
				{autoscan.Scan{Folder: "1", File: "1", Retries: 1}, testTime.Add(5 * time.Minute)},
				{autoscan.Scan{Folder: "1", File: "2", Retries: 3}, testTime.Add(5 * time.Minute)},
				{autoscan.Scan{Folder: "2", File: "1", Retries: 0}, testTime.Add(0 * time.Minute)},
			},
		},
		{
			Name:    "Retry older than max value should get deleted",
			Folder:  "1",
			Retries: 2,
			Scans: []autoscan.Scan{
				{Folder: "1", File: "1", Retries: 1},
				{Folder: "1", File: "2", Retries: 2},
			},
			Want: []ScanWithTime{
				{autoscan.Scan{Folder: "1", File: "1", Retries: 2}, testTime.Add(5 * time.Minute)},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			store, err := newDatastore(":memory:")
			if err != nil {
				t.Fatal(err)
			}

			now = func() time.Time {
				return testTime
			}

			err = store.Upsert(tc.Scans)
			if err != nil {
				t.Fatal(err)
			}

			now = func() time.Time {
				return testTime.Add(5 * time.Minute)
			}

			err = store.Retry(tc.Folder, tc.Retries)
			if !errors.Is(err, tc.Err) {
				t.Fatal(err)
			}

			scans, err := store.GetAllWithTime()
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(scans, tc.Want) {
				t.Log(scans)
				t.Log(tc.Want)
				t.Errorf("Scans do not match")
			}
		})
	}
}

func TestDelete(t *testing.T) {
	type Test struct {
		Name   string
		Scans  []autoscan.Scan
		Delete []autoscan.Scan
		Want   []autoscan.Scan
	}

	var testCases = []Test{
		{
			Name: "Only deletes specific file, not all files in folder nor files in other folders",
			Scans: []autoscan.Scan{
				{Folder: "1", File: "1"},
				{Folder: "1", File: "2"},
				{Folder: "2", File: "1"},
			},
			Delete: []autoscan.Scan{
				{Folder: "1", File: "1"},
			},
			Want: []autoscan.Scan{
				{Folder: "1", File: "2"},
				{Folder: "2", File: "1"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			store, err := newDatastore(":memory:")
			if err != nil {
				t.Fatal(err)
			}

			err = store.Upsert(tc.Scans)
			if err != nil {
				t.Fatal(err)
			}

			err = store.Delete(tc.Delete)
			if err != nil {
				t.Fatal(err)
			}

			scans, err := store.GetAll()
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
