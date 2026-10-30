-- Instance settings (M2c Task 3): the typed reads of the key-value
-- configuration table. The trash retention is the first consumer; the
-- value column is jsonb so every setting reads as raw JSON and the
-- typed interpretation (integer, boolean, string) lives with the
-- caller's accessor.

-- name: GetInstanceSetting :one
-- The raw jsonb value of one setting key. No row means the setting
-- was never written; the accessor's default is the answer then.
SELECT value FROM instance_settings
WHERE key = $1;
