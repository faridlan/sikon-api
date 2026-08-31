DROP TABLE IF EXISTS attendances;

DROP INDEX IF EXISTS idx_workers_user_id;

ALTER TABLE workers
DROP COLUMN IF EXISTS daily_rate,
DROP COLUMN IF EXISTS user_id;