-- Tambahkan kolom deleted_at beserta index-nya ke semua tabel master dan transaksi

ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

ALTER TABLE categories ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_categories_deleted_at ON categories(deleted_at);

ALTER TABLE customers ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_customers_deleted_at ON customers(deleted_at);

ALTER TABLE products ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_products_deleted_at ON products(deleted_at);

ALTER TABLE bank_accounts ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_bank_accounts_deleted_at ON bank_accounts(deleted_at);

ALTER TABLE spec_templates ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_spec_templates_deleted_at ON spec_templates(deleted_at);

ALTER TABLE orders ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_orders_deleted_at ON orders(deleted_at);

ALTER TABLE order_items ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_order_items_deleted_at ON order_items(deleted_at);

ALTER TABLE payments ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_payments_deleted_at ON payments(deleted_at);