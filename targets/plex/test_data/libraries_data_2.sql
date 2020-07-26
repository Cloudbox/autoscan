-- data
INSERT INTO "section_locations" ("id", "library_section_id", "root_path", "available", "scanned_at", "created_at", "updated_at") VALUES
('10', 10, '/data/Music', '1', '2020-06-24 09:34:38', '2017-11-30 03:46:04', '2020-06-24 09:35:25'),
('12', 12, '/data/4K/Movies', '1', '2020-07-04 06:10:51', '2018-05-06 01:57:32', '2020-07-04 06:10:51');

INSERT INTO "library_sections" ("id", "library_id", "name", "name_sort", "section_type", "language", "agent", "scanner", "user_thumb_url", "user_art_url", "user_theme_music_url", "public", "created_at", "updated_at", "scanned_at", "display_secondary_level", "user_fields", "query_xml", "query_type", "uuid", "changed_at", "content_changed_at") VALUES
('10', NULL, 'Music', '', 8, 'en', 'tv.plex.agents.music', 'Plex Music', '', '', '', NULL, '2017-11-30 03:46:04', '2020-06-24 09:38:35', '2020-06-24 09:35:25', NULL, 'pr%3Ahidden=1&pr%3ArespectTags=1&pv%3AfirstLoudnessScan=0&pv%3AlastAddedAt=1592984113', '', NULL, '68dd5a9f-b3eb-4439-8ade-06683a4f5cf9', '45659387', '45659199'),
('12', NULL, '4K - Movies', '', 1, 'en', 'com.plexapp.agents.imdb', 'Plex Movie Scanner', '', '', '', NULL, '2018-05-06 01:57:32', '2020-06-24 09:32:38', '2020-06-24 09:31:42', NULL, 'pr%3AcollectionMode=1&pr%3AenableBIFGeneration=0&pr%3Ahidden=1&pv%3AlastAddedAt=1592885056', '', NULL, '1820f866-ca5f-4732-84d7-705f799bce22', '45659148', '47045075');
