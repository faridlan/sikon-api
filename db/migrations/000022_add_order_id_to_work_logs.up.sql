-- Tambahkan relasi order_id untuk mencatat pengerjaan borongan per konsumen/order spesifik
ALTER TABLE work_logs
ADD COLUMN order_id UUID NULL REFERENCES orders (id) ON DELETE SET NULL;

CREATE INDEX idx_work_logs_order_id ON work_logs (order_id);