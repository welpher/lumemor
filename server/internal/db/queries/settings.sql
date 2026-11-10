-- Instance settings: the typed reads of the key-value
-- configuration table. The trash retention is the first consumer; the
-- value column is jsonb so every setting reads as raw JSON and the
-- typed interpretation (integer, boolean, string) lives with the
-- caller's accessor.

-- name: GetInstanceSetting :one
-- The raw jsonb value of one setting key. No row means the setting
-- was never written; the accessor's default is the answer then.
SELECT value FROM instance_settings
WHERE key = $1;

-- name: SetInstanceSetting :exec
-- The typed writer behind every setting's save: the value lands as a
-- jsonb string and the row's updated_at moves with the write. The
-- upsert arm serves the save-over-existing flow; the insert arm
-- serves the first write.
INSERT INTO instance_settings (key, value)
VALUES (sqlc.arg('key'), to_jsonb(sqlc.arg('value')::text))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();

-- name: SetInstanceSettingRaw :exec
-- The jsonb-native writer (the settings feature): the value is
-- already JSON -- a number for the trash retention, an object for
-- the transcoding profile -- and lands verbatim, so the typed
-- readers' unmarshal sees the JSON value they expect (a text-cast
-- writer would store a JSON string and every int/object reader
-- would fall back to its default). The upsert arm serves the
-- save-over-existing flow; the insert arm serves the first write.
INSERT INTO instance_settings (key, value)
VALUES (sqlc.arg('key'), sqlc.arg('value')::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
