CREATE TABLE IF NOT EXISTS batch_pos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    name VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    quota INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Menambahkan relasi ke tabel orders
ALTER TABLE orders ADD COLUMN batch_po_id UUID;

ALTER TABLE orders
ADD CONSTRAINT fk_orders_batch_pos FOREIGN KEY (batch_po_id) REFERENCES batch_pos (id) ON DELETE RESTRICT;