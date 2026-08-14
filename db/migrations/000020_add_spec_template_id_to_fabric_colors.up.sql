-- 1. Tambah kolom spec_template_id (NULLABLE agar warna bisa terikat ke SpecTemplate)
ALTER TABLE fabric_colors
ADD COLUMN spec_template_id UUID REFERENCES spec_templates (id) ON DELETE CASCADE;

-- 2. Ubah fabric_id menjadi NULLABLE (karena warna di SpecTemplate belum/tidak punya fabric_id)
ALTER TABLE fabric_colors ALTER COLUMN fabric_id DROP NOT NULL;

-- 3. Tambahkan index untuk mempercepat query preload warna berdasarkan spec_template_id
CREATE INDEX idx_fabric_colors_spec_template_id ON fabric_colors (spec_template_id);