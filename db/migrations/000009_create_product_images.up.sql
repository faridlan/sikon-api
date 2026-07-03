-- Hapus kolom image_url dari tabel products
ALTER TABLE products DROP COLUMN IF EXISTS image_url;

-- Buat tabel relasi untuk menampung banyak gambar
CREATE TABLE product_images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    image_url VARCHAR(255) NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk mempercepat query saat mencari gambar milik suatu produk
CREATE INDEX idx_product_images_product_id ON product_images (product_id);