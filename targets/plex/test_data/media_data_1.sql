-- data
INSERT INTO "directories" ("id", "library_section_id", "parent_directory_id", "path", "created_at", "updated_at", "deleted_at") VALUES
('1', 1, NULL, '', '2017-07-31 21:38:34', '2020-06-26 12:03:45', NULL),
('3', 1, 1, 'Æon Flux (2005)', '2017-07-31 21:38:37', '2020-04-25 06:16:37', NULL),
('5', 1, 1, '10 Cloverfield Lane (2016)', '2017-07-31 21:38:41', '2020-04-25 04:14:47', NULL);

INSERT INTO media_parts(extra_data, deleted_at, updated_at, created_at, duration, size, [index], file, open_subtitle_hash, hash, directory_id, media_item_id, id) VALUES
('ma%3Acontainer=mkv&ma%3AvideoProfile=high', NULL, '2016-06-05 10:10:29', '2016-06-05 10:10:29', 6214816, 24116451791, NULL, '/data/Movies/10 Cloverfield Lane (2016)/10 Cloverfield Lane (2016) - Remux-1080p.x264.TrueHD-SiDMUX.mkv', '8a4c08cc9ed26c87', '71eab61c009b15661e545c21d3afeb9783238912', 5, 82884, 83073),
('ma%3AaudioProfile=ma&ma%3Acontainer=mkv&ma%3AvideoProfile=high', NULL, '2017-02-23 23:33:42', '2017-02-23 23:33:42', 5571584, 24733076291, NULL, '/data/Movies/Æon Flux (2005)/Æon Flux (2005) - Remux-1080p.x264.DTS-HD.MA-HD.mkv', '60f7de13f9d8d087', '5c736a5ab8d58d2f34f3102867847cc66f6a3a7e', 3, 49794, 49967);

INSERT INTO media_items (color_trc, ends_at, begins_at, channel_id, proxy_type, extra_data, sample_aspect_ratio, media_analysis_version, deleted_at, optimized_for_streaming, updated_at, created_at, settings, display_offset, hints, source, interlaced, audio_channels, frames_per_second, display_aspect_ratio, audio_codec, video_codec, container, bitrate, duration, size, height, width, type_id, metadata_item_id, section_location_id, library_section_id, id) VALUES
(NULL, NULL, NULL, NULL, NULL, 'ma%3AvideoProfile=high', NULL, 6, NULL, NULL, '2016-06-05 10:10:29', '2016-06-05 10:10:29', '', 0, 'name=10%20Cloverfield%20Lane&year=2016', '', NULL, 8, 23.97602462768555, 1.77777779102325, 'truehd', 'h264', 'mkv', 31043817, 6214816, 24116451791, 1080, 1920, NULL, 5, 1, 1, 82884),
(NULL, NULL, NULL, NULL, NULL, 'ma%3AaudioProfile=ma&ma%3AvideoProfile=high', NULL, 5, NULL, NULL, '2017-02-23 23:33:42', '2017-02-23 23:33:42', '', 0, 'name=%C3%A6on%20Flux&year=2005', '', NULL, 6, 23.97602462768555, 1.77777779102325, 'dca', 'h264', 'mkv', 35513170, 5571584, 24733076291, 1080, 1920, NULL, 3, 1, 1, 49794);