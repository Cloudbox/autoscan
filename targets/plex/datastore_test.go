package plex

import (
	"reflect"
	"testing"

	// database driver
	_ "github.com/mattn/go-sqlite3"
)

func setupTest(t *testing.T) *Datastore {
	t.Helper()

	ds, err := New(":memory:")
	if err != nil {
		t.Fatal("Could not create datastore")
	}

	if _, err := ds.db.Exec(sqlTestSchema); err != nil {
		t.Fatalf("Could not create test schema: %v", err)
	}

	if _, err := ds.db.Exec(sqlTestData); err != nil {
		t.Fatalf("Could not create test data: %v", err)
	}

	return ds
}

func TestDatastore_Libraries(t *testing.T) {
	store := setupTest(t)

	libraries, err := store.Libraries()
	if err != nil {
		t.Fatalf("Error getting libraries: %v", err)
	}

	if len(libraries) < 4 {
		t.Fatalf("Library counts do not match, expected: 4, got: %d", len(libraries))
	}

	expected := []Library{
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
	}

	if !reflect.DeepEqual(libraries, expected) {
		t.Log(libraries)
		t.Log(expected)
		t.Errorf("Libraries do not match")
	}
}

const sqlTestSchema string = `
PRAGMA foreign_keys=ON;

CREATE TABLE "section_locations" ("id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, "library_section_id" integer, "root_path" varchar(255), "available" boolean DEFAULT 't', "scanned_at" datetime, "created_at" datetime, "updated_at" datetime);
CREATE TABLE "library_sections" ("id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, "library_id" integer, "name" varchar(255), "name_sort" varchar(255) COLLATE NOCASE, "section_type" integer, "language" varchar(255), "agent" varchar(255), "scanner" varchar(255), "user_thumb_url" varchar(255), "user_art_url" varchar(255), "user_theme_music_url" varchar(255), "public" boolean, "created_at" datetime, "updated_at" datetime, "scanned_at" datetime, "display_secondary_level" boolean, "user_fields" varchar(255), "query_xml" text, "query_type" integer, "uuid" varchar(255), "changed_at" integer(8) DEFAULT 0, 'content_changed_at' integer(8) default '0');
`
const sqlTestData = `
INSERT INTO "section_locations" ("id", "library_section_id", "root_path", "available", "scanned_at", "created_at", "updated_at") VALUES
('1', 1, '/data/Movies', '1', '2020-07-04 12:25:59', '2017-07-31 21:38:34', '2020-07-04 12:25:59'),
('2', 2, '/data/TV', '1', '2020-07-04 13:22:52', '2017-08-01 23:20:31', '2020-07-04 13:22:52'),
('10', 10, '/data/Music', '1', '2020-06-24 09:34:38', '2017-11-30 03:46:04', '2020-06-24 09:35:25'),
('12', 12, '/data/4K/Movies', '1', '2020-07-04 06:10:51', '2018-05-06 01:57:32', '2020-07-04 06:10:51');

INSERT INTO "library_sections" ("id", "library_id", "name", "name_sort", "section_type", "language", "agent", "scanner", "user_thumb_url", "user_art_url", "user_theme_music_url", "public", "created_at", "updated_at", "scanned_at", "display_secondary_level", "user_fields", "query_xml", "query_type", "uuid", "changed_at", "content_changed_at") VALUES
('1', NULL, 'Movies', '', 1, 'en', 'com.plexapp.agents.imdb', 'Plex Movie Scanner', '', '', '', NULL, '2017-07-31 21:38:34', '2020-06-26 12:11:46', '2020-06-26 12:07:43', NULL, 'pr%3AcollectionMode=1&pr%3AenableBIFGeneration=0&pr%3AenableCinemaTrailers=0&pv%3AlastAddedAt=1593155849', '', NULL, 'b755029d-39a1-421f-8850-b4f11dde469e', '46005545', '47050956'),
('2', NULL, 'TV', '', 2, 'en', 'com.plexapp.agents.thetvdb', 'Plex Series Scanner', '', '', '', NULL, '2017-08-01 23:20:31', '2020-06-26 12:19:56', '2020-06-24 09:41:56', NULL, 'pr%3AenableBIFGeneration=0&pv%3AlastAddedAt=1593162644', '', NULL, '5ac23cdc-5321-4aca-9d50-63f6e58b402e', '46005877', '47051099'),
('10', NULL, 'Music', '', 8, 'en', 'tv.plex.agents.music', 'Plex Music', '', '', '', NULL, '2017-11-30 03:46:04', '2020-06-24 09:38:35', '2020-06-24 09:35:25', NULL, 'pr%3Ahidden=1&pr%3ArespectTags=1&pv%3AfirstLoudnessScan=0&pv%3AlastAddedAt=1592984113', '', NULL, '68dd5a9f-b3eb-4439-8ade-06683a4f5cf9', '45659387', '45659199'),
('12', NULL, '4K - Movies', '', 1, 'en', 'com.plexapp.agents.imdb', 'Plex Movie Scanner', '', '', '', NULL, '2018-05-06 01:57:32', '2020-06-24 09:32:38', '2020-06-24 09:31:42', NULL, 'pr%3AcollectionMode=1&pr%3AenableBIFGeneration=0&pr%3Ahidden=1&pv%3AlastAddedAt=1592885056', '', NULL, '1820f866-ca5f-4732-84d7-705f799bce22', '45659148', '47045075');
`
