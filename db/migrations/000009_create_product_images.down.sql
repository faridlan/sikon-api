-- 1. Hapus indeks yang dibuat sebelumnya
DROP INDEX IF EXISTS idx_product_images_product_id;

-- 2. Hapus tabel relasi product_images
DROP TABLE IF EXISTS product_images;

-- 3. Kembalikan kolom image_url ke tabel products
ALTER TABLE products ADD COLUMN IF NOT EXISTS image_url VARCHAR(255);