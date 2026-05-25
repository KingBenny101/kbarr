SET search_path = core;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'episodes'
          AND column_name = 'ani_db_id'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'episodes'
          AND column_name = 'anidb_id'
    ) THEN
        ALTER TABLE episodes RENAME COLUMN ani_db_id TO anidb_id;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'monitors'
          AND column_name = 'ani_db_id'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'monitors'
          AND column_name = 'anidb_id'
    ) THEN
        ALTER TABLE monitors RENAME COLUMN ani_db_id TO anidb_id;
    END IF;
END
$$;