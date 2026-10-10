-- Baseline schema for the app. Auth lives in Neon (neon_auth schema) and is
-- never touched here. A legacy `lookups` table may exist; it is ignored.
--
-- Written with IF NOT EXISTS throughout so applying on a database that
-- GORM AutoMigrate already created is a safe no-op. Never edit this file
-- after it has been applied anywhere; new changes go in new migrations.

CREATE TABLE IF NOT EXISTS links (
  id bigserial PRIMARY KEY,
  original_url text NOT NULL,
  short_code varchar(10) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- GORM names these the same way, so on AutoMigrate-created databases
-- each statement below is a no-op.
CREATE UNIQUE INDEX IF NOT EXISTS uni_links_short_code ON links (short_code);
CREATE INDEX IF NOT EXISTS idx_links_short_code ON links (short_code);
CREATE INDEX IF NOT EXISTS idx_links_original_url ON links (original_url);
