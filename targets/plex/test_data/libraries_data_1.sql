-- data
INSERT INTO "section_locations" ("id", "library_section_id", "root_path", "available", "scanned_at", "created_at", "updated_at") VALUES
('1', 1, '/data/Movies', '1', '2020-07-04 12:25:59', '2017-07-31 21:38:34', '2020-07-04 12:25:59'),
('2', 2, '/data/TV', '1', '2020-07-04 13:22:52', '2017-08-01 23:20:31', '2020-07-04 13:22:52');

INSERT INTO "library_sections" ("id", "library_id", "name", "name_sort", "section_type", "language", "agent", "scanner", "user_thumb_url", "user_art_url", "user_theme_music_url", "public", "created_at", "updated_at", "scanned_at", "display_secondary_level", "user_fields", "query_xml", "query_type", "uuid", "changed_at", "content_changed_at") VALUES
('1', NULL, 'Movies', '', 1, 'en', 'com.plexapp.agents.imdb', 'Plex Movie Scanner', '', '', '', NULL, '2017-07-31 21:38:34', '2020-06-26 12:11:46', '2020-06-26 12:07:43', NULL, 'pr%3AcollectionMode=1&pr%3AenableBIFGeneration=0&pr%3AenableCinemaTrailers=0&pv%3AlastAddedAt=1593155849', '', NULL, 'b755029d-39a1-421f-8850-b4f11dde469e', '46005545', '47050956'),
('2', NULL, 'TV', '', 2, 'en', 'com.plexapp.agents.thetvdb', 'Plex Series Scanner', '', '', '', NULL, '2017-08-01 23:20:31', '2020-06-26 12:19:56', '2020-06-24 09:41:56', NULL, 'pr%3AenableBIFGeneration=0&pv%3AlastAddedAt=1593162644', '', NULL, '5ac23cdc-5321-4aca-9d50-63f6e58b402e', '46005877', '47051099');

