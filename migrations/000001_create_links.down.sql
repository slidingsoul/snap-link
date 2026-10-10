-- Rollback for 000001. Dev/CI only; never run down on production blindly.
DROP INDEX IF EXISTS idx_links_original_url;
DROP INDEX IF EXISTS idx_links_short_code;
DROP INDEX IF EXISTS uni_links_short_code;
DROP TABLE IF EXISTS links;
