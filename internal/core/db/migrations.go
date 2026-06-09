package db

import (
	"context"
	"fmt"
)

// runMigrations applies schema changes that cannot be expressed with CreateTable IfNotExists.
// All statements are idempotent and safe to re-run on every start.
func runMigrations(ctx context.Context) error {
	stmts := []string{
		// media
		`ALTER TABLE media ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'anidb'`,
		`ALTER TABLE media ADD COLUMN IF NOT EXISTS source_id TEXT NOT NULL DEFAULT ''`,
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='media' AND column_name='aid') THEN
				UPDATE media SET source_id = aid::text WHERE source_id = '' AND aid IS NOT NULL;
			END IF;
		END $$`,
		`ALTER TABLE media DROP COLUMN IF EXISTS aid`,

		// detaileds
		`ALTER TABLE detaileds ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'anidb'`,
		`ALTER TABLE detaileds ADD COLUMN IF NOT EXISTS source_id TEXT NOT NULL DEFAULT ''`,
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='detaileds' AND column_name='aid') THEN
				UPDATE detaileds SET source_id = aid::text WHERE source_id = '' AND aid IS NOT NULL;
			END IF;
		END $$`,
		`ALTER TABLE detaileds DROP COLUMN IF EXISTS aid`,

		// episodes
		`ALTER TABLE episodes ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'anidb'`,
		`ALTER TABLE episodes ADD COLUMN IF NOT EXISTS external_id TEXT`,
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='episodes' AND column_name='anidb_id') THEN
				UPDATE episodes SET external_id = anidb_id WHERE external_id IS NULL AND anidb_id IS NOT NULL;
			END IF;
		END $$`,
		`ALTER TABLE episodes DROP COLUMN IF EXISTS anidb_id`,

		// monitors
		`ALTER TABLE monitors ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'anidb'`,
		`ALTER TABLE monitors ADD COLUMN IF NOT EXISTS external_id TEXT`,
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='monitors' AND column_name='anidb_id') THEN
				UPDATE monitors SET external_id = anidb_id WHERE external_id IS NULL AND anidb_id IS NOT NULL;
			END IF;
		END $$`,
		`ALTER TABLE monitors DROP COLUMN IF EXISTS anidb_id`,

		// download_queue
		`ALTER TABLE download_queue ADD COLUMN IF NOT EXISTS progress REAL DEFAULT 0`,

		// download_queue extra columns added in later session
		`ALTER TABLE download_queue ADD COLUMN IF NOT EXISTS torrent_name TEXT`,
		`ALTER TABLE download_queue ADD COLUMN IF NOT EXISTS indexer TEXT`,
		`ALTER TABLE download_queue ADD COLUMN IF NOT EXISTS size BIGINT`,
		`ALTER TABLE download_queue ADD COLUMN IF NOT EXISTS seeders INT`,
		`ALTER TABLE download_queue ADD COLUMN IF NOT EXISTS progress_updated_at TIMESTAMPTZ`,

		// monitors: add available column and migrate old 'available' status rows (runs once)
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='monitors' AND column_name='available') THEN
				ALTER TABLE monitors ADD COLUMN available boolean NOT NULL DEFAULT false;
				UPDATE monitors SET available = true WHERE status = 'available';
				UPDATE monitors SET status = 'downloaded' WHERE status = 'available' AND EXISTS (SELECT 1 FROM download_queue WHERE monitor_id = monitors.id AND status = 'completed' AND deleted_at IS NULL);
				UPDATE monitors SET status = 'unmonitored' WHERE status = 'available' AND NOT EXISTS (SELECT 1 FROM download_queue WHERE monitor_id = monitors.id AND status = 'completed' AND deleted_at IS NULL);
			END IF;
		END $$`,

		// monitors: add monitored boolean and rename status values (runs once)
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='monitors' AND column_name='monitored') THEN
				ALTER TABLE monitors ADD COLUMN monitored boolean NOT NULL DEFAULT false;
				UPDATE monitors SET monitored = true WHERE status NOT IN ('unmonitored') AND deleted_at IS NULL;
				UPDATE monitors SET status = 'pending' WHERE status = 'monitored';
				UPDATE monitors SET status = 'pending' WHERE status = 'unmonitored';
				UPDATE monitors SET status = 'pending' WHERE status = 'available';
			END IF;
		END $$`,

		// media: add media_folder — canonical sanitized directory name used by downloader and availability check
		`ALTER TABLE media ADD COLUMN IF NOT EXISTS media_folder TEXT NOT NULL DEFAULT ''`,
		// Backfill existing rows: replace filesystem-invalid chars with underscore, trim whitespace
		`UPDATE media SET media_folder = TRIM(REGEXP_REPLACE(COALESCE(title, ''), E'[/\\\\:*?"<>|]', '_', 'g')) WHERE media_folder = ''`,
	}

	for _, stmt := range stmts {
		if _, err := DB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed [%.80s]: %w", stmt, err)
		}
	}
	return nil
}
