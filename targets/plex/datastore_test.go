package plex

import (
	"errors"
	"io/ioutil"
	"reflect"
	"testing"
	// database driver
	_ "github.com/mattn/go-sqlite3"
)

func setupTest(t *testing.T) *Datastore {
	t.Helper()

	ds, err := NewDatastore(":memory:")
	if err != nil {
		t.Fatal("Could not create datastore")
	}

	if _, err := ds.db.Exec(`PRAGMA foreign_keys=ON;`); err != nil {
		t.Fatal("Could not prepare datastore")
	}

	return ds
}

func setupDatabase(t *testing.T, ds *Datastore, paths []string) {
	if len(paths) == 0 {
		return
	}

	for _, path := range paths {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatalf("Could not read database test file %q: %v", path, err)
		}

		if _, err := ds.db.Exec(string(b)); err != nil {
			t.Fatalf("Could not exec database test file data %q: %v", path, err)
		}
	}
}

func TestDatastore_Media(t *testing.T) {
	type expect struct {
		item *MediaPart
		err  error
	}
	type test struct {
		sql  []string
		item string
		want expect
	}

	tests := []test{
		{
			sql:  []string{"test_data/media_schema.sql", "test_data/media_data_1.sql"},
			item: "/data/Movies/10 Cloverfield Lane (2016)/10 Cloverfield Lane (2016) - Remux-1080p.x264.TrueHD-SiDMUX.mkv",
			want: expect{
				item: &MediaPart{
					ID:          83073,
					DirectoryID: 5,
					File:        "/data/Movies/10 Cloverfield Lane (2016)/10 Cloverfield Lane (2016) - Remux-1080p.x264.TrueHD-SiDMUX.mkv",
					Size:        24116451791,
				},
				err: nil,
			},
		},
		{
			item: "no such file.mkv",
			want: expect{
				err: ErrDatabaseRowNotFound,
			},
		},
	}

	store := setupTest(t)

	for _, tc := range tests {
		// prepare
		setupDatabase(t, store, tc.sql)

		mp, err := store.MediaPartByFile(tc.item)
		if err != nil {
			if errors.Is(err, tc.want.err) {
				// expected this error
				continue
			} else if tc.want.err != nil {
				// unexpected error
				t.Log(tc.want.err)
				t.Log(err)
				t.Fatal("Error does not match")
			}

			t.Fatalf("Error selecting media part: %v", err)
		}

		if !reflect.DeepEqual(mp, tc.want.item) {
			t.Log(mp)
			t.Log(tc.want)
			t.Errorf("MediaPart does not match")
		}
	}
}

func TestDatastore_Libraries(t *testing.T) {
	type test struct {
		sql  []string
		size int
		want []Library
	}

	tests := []test{
		{
			sql:  []string{"test_data/libraries_schema.sql", "test_data/libraries_data_1.sql"},
			size: 2,
			want: []Library{
				{
					ID:   1,
					Type: Movie,
					Name: "Movies",
					Path: "/data/Movies",
				},
				{
					ID:   2,
					Type: TV,
					Name: "TV",
					Path: "/data/TV",
				},
			},
		},
		{
			sql:  []string{"test_data/libraries_data_2.sql"},
			size: 4,
			want: []Library{
				{
					ID:   1,
					Type: Movie,
					Name: "Movies",
					Path: "/data/Movies",
				},
				{
					ID:   2,
					Type: TV,
					Name: "TV",
					Path: "/data/TV",
				},
				{
					ID:   10,
					Type: Music,
					Name: "Music",
					Path: "/data/Music",
				},
				{
					ID:   12,
					Type: Movie,
					Name: "4K - Movies",
					Path: "/data/4K/Movies",
				},
			},
		},
	}

	store := setupTest(t)

	for _, tc := range tests {
		// prepare
		setupDatabase(t, store, tc.sql)

		// test
		libraries, err := store.Libraries()
		if err != nil {
			t.Fatalf("Error getting libraries: %v", err)
		}

		if len(libraries) != tc.size {
			t.Fatalf("Library counts do not match, expected: %d, got: %d", tc.size, len(libraries))
		}

		if !reflect.DeepEqual(libraries, tc.want) {
			t.Log(libraries)
			t.Log(tc.want)
			t.Errorf("Libraries do not match")
		}
	}
}
