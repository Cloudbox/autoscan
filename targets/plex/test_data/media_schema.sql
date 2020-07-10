-- schema
CREATE TABLE "directories" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "library_section_id" integer,
    "parent_directory_id" integer,
    "path" varchar(255),
    "created_at" datetime,
    "updated_at" datetime,
    "deleted_at" datetime
);

CREATE TABLE "media_items"
(
    "id"                      INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "library_section_id"      integer,
    "section_location_id"     integer,
    "metadata_item_id"        integer,
    "type_id"                 integer,
    "width"                   integer,
    "height"                  integer,
    "size"                    integer(8),
    "duration"                integer,
    "bitrate"                 integer,
    "container"               varchar(255),
    "video_codec"             varchar(255),
    "audio_codec"             varchar(255),
    "display_aspect_ratio"    float,
    "frames_per_second"       float,
    "audio_channels"          integer,
    "interlaced"              boolean,
    "source"                  varchar(255),
    "hints"                   varchar(255),
    "display_offset"          integer,
    "settings"                varchar(255),
    "created_at"              datetime,
    "updated_at"              datetime,
    "optimized_for_streaming" boolean,
    "deleted_at"              datetime,
    "media_analysis_version"  integer DEFAULT 0,
    "sample_aspect_ratio"     float,
    "extra_data"              varchar(255),
    "proxy_type"              integer,
    "channel_id"              integer,
    "begins_at"               datetime,
    "ends_at"                 datetime,
    "color_trc"               varchar(255)
);

CREATE TABLE "media_parts"
(
    "id"                 INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    "media_item_id"      integer,
    "directory_id"       integer,
    "hash"               varchar(255),
    "open_subtitle_hash" varchar(255),
    "file"               varchar(255),
    "index"              integer,
    "size"               integer(8),
    "duration"           integer,
    "created_at"         datetime,
    "updated_at"         datetime,
    "deleted_at"         datetime,
    "extra_data"         varchar(255)
);