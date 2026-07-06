-- Tambahkan kolom image_url ke tabel categories
ALTER TABLE categories ADD COLUMN IF NOT EXISTS image_url VARCHAR(255);
