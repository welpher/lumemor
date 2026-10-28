-- +goose Up
-- Asset domain: assets, EXIF details, and the place reference data used for
-- city grouping and reverse geocoding.
--
-- places and place_names hold build-time GeoNames reference data, so they
-- carry no audit columns and place_names deliberately has no foreign key to
-- places: geoname_id values may also address admin1 entities that never get
-- a places row.

CREATE TABLE places (
    id           integer PRIMARY KEY, -- GeoNames geonameid
    name         text NOT NULL,
    country_code text NOT NULL,
    admin1_code  text NOT NULL,
    timezone     text NOT NULL,
    lat          double precision NOT NULL CHECK (lat BETWEEN -90 AND 90),
    lon          double precision NOT NULL CHECK (lon BETWEEN -180 AND 180)
);

CREATE TABLE place_names (
    geoname_id integer NOT NULL,
    locale     text NOT NULL,
    name       text NOT NULL,
    PRIMARY KEY (geoname_id, locale)
);

CREATE TABLE assets (
    id                   uuid PRIMARY KEY,
    owner_id             uuid NOT NULL REFERENCES users (id),
    storage_kind         storage_kind NOT NULL,
    file_path            text NOT NULL,
    checksum             text NOT NULL,
    original_filename    text NOT NULL,
    file_size            bigint NOT NULL,
    mime_type            text NOT NULL,
    media_kind           media_kind NOT NULL,
    live_photo_video_id  uuid REFERENCES assets (id) ON DELETE SET NULL,
    is_component         boolean NOT NULL DEFAULT false,
    width                integer NOT NULL,
    height               integer NOT NULL,
    duration_ms          integer,
    taken_at             timestamptz NOT NULL,
    taken_at_source      taken_at_source NOT NULL,
    place_id             integer REFERENCES places (id) ON DELETE SET NULL,
    blurhash             text,
    phash                text,
    is_hidden            boolean NOT NULL DEFAULT false,
    description          text NOT NULL DEFAULT '',
    deleted_at           timestamptz,
    original_path        text,
    trash_path           text,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

-- Soft-deleted assets release their checksum so a re-upload of the same
-- file can be flagged as "already in trash" instead of hard-failing.
CREATE UNIQUE INDEX assets_owner_checksum_live_uq
    ON assets (owner_id, checksum)
    WHERE deleted_at IS NULL;

-- Timeline keyset pagination key, restricted to the rows the timeline can
-- serve.
CREATE INDEX assets_timeline_idx
    ON assets (taken_at DESC, id DESC)
    WHERE deleted_at IS NULL AND NOT is_hidden;

CREATE INDEX assets_owner_idx ON assets (owner_id);

CREATE TABLE exif (
    asset_id     uuid PRIMARY KEY REFERENCES assets (id) ON DELETE CASCADE,
    exif         jsonb NOT NULL,
    camera_make  text,
    camera_model text,
    lens         text,
    iso          integer,
    aperture     real,
    shutter      text,
    focal        real,
    lat          double precision CHECK (lat BETWEEN -90 AND 90),
    lon          double precision CHECK (lon BETWEEN -180 AND 180),
    alt          real,
    tz_offset    text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS exif;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS place_names;
DROP TABLE IF EXISTS places;
