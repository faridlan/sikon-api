DROP INDEX IF EXISTS idx_work_logs_order_id;

ALTER TABLE work_logs DROP COLUMN IF EXISTS order_id;