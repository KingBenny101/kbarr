CREATE SCHEMA IF NOT EXISTS core;
SET search_path = core;

CREATE TABLE IF NOT EXISTS media (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    title TEXT,
    aid BIGINT,
    poster_url TEXT
);
CREATE INDEX IF NOT EXISTS idx_media_deleted_at ON media (deleted_at);
CREATE INDEX IF NOT EXISTS idx_media_aid_deleted_at ON media (aid, deleted_at);

CREATE TABLE IF NOT EXISTS detaileds (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    title TEXT,
    aid BIGINT,
    library_id BIGINT,
    alternate_titles TEXT,
    description TEXT,
    release_date TEXT,
    genres TEXT,
    poster_url TEXT,
    total_episodes BIGINT,
    total_seasons BIGINT
);
CREATE INDEX IF NOT EXISTS idx_detaileds_deleted_at ON detaileds (deleted_at);
CREATE INDEX IF NOT EXISTS idx_detaileds_library_id_deleted_at ON detaileds (library_id, deleted_at);

CREATE TABLE IF NOT EXISTS episodes (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    detailed_id BIGINT,
    anidb_id TEXT,
    type BIGINT,
    ep_no TEXT,
    title TEXT,
    air_date TEXT,
    CONSTRAINT fk_episodes_detailed FOREIGN KEY (detailed_id) REFERENCES detaileds(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_episodes_deleted_at ON episodes (deleted_at);
CREATE INDEX IF NOT EXISTS idx_episodes_detailed_id_deleted_at ON episodes (detailed_id, deleted_at);

CREATE TABLE IF NOT EXISTS monitors (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    library_id BIGINT,
    title TEXT,
    episode_title TEXT,
    season BIGINT,
    episode_number BIGINT,
    is_episode BOOLEAN,
    is_season BOOLEAN,
    anidb_id TEXT,
    status TEXT DEFAULT 'monitored'
);
CREATE INDEX IF NOT EXISTS idx_monitors_deleted_at ON monitors (deleted_at);
CREATE INDEX IF NOT EXISTS idx_monitors_library_id_deleted_at ON monitors (library_id, deleted_at);

CREATE TABLE IF NOT EXISTS search_queues (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    library_id BIGINT,
    title TEXT,
    episode_title TEXT,
    season BIGINT,
    episode_number BIGINT,
    is_episode BOOLEAN,
    is_season BOOLEAN,
    status TEXT DEFAULT 'pending'
);
CREATE INDEX IF NOT EXISTS idx_search_queues_deleted_at ON search_queues (deleted_at);
CREATE INDEX IF NOT EXISTS idx_search_queues_library_id_deleted_at ON search_queues (library_id, deleted_at);