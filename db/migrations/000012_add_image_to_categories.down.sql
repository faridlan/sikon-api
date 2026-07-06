-- Hapus kolom image_url dari tabel categories
ALTER TABLE categories DROP COLUMN IF EXISTS image_url;
