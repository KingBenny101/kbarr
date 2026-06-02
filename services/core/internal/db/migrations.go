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
	}

	for _, stmt := range stmts {
		if _, err := DB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed [%.80s]: %w", stmt, err)
		}
	}
	return nil
}
