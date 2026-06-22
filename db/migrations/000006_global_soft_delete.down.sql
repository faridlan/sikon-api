-- Hapus index dan kolom deleted_at dari semua tabel

DROP INDEX IF EXISTS idx_users_deleted_at;
ALTER TABLE users DROP COLUMN deleted_at;

DROP INDEX IF EXISTS idx_categories_deleted_at;
ALTER TABLE categories DROP COLUMN deleted_at;

DROP INDEX IF EXISTS idx_customers_deleted_at;
ALTER TABLE customers DROP COLUMN deleted_at;

DROP INDEX IF EXISTS idx_products_deleted_at;
ALTER TABLE products DROP COLUMN deleted_at;

DROP INDEX IF EXISTS idx_bank_accounts_deleted_at;
ALTER TABLE bank_accounts DROP COLUMN deleted_at;

DROP INDEX IF EXISTS idx_spec_templates_deleted_at;
ALTER TABLE spec_templates DROP COLUMN deleted_at;

DROP INDEX IF EXISTS idx_orders_deleted_at;
ALTER TABLE orders DROP COLUMN deleted_at;

DROP INDEX IF EXISTS idx_order_items_deleted_at;
ALTER TABLE order_items DROP COLUMN deleted_at;

DROP INDEX IF EXISTS idx_payments_deleted_at;
ALTER TABLE payments DROP COLUMN deleted_at;