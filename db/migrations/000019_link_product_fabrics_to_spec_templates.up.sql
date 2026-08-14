-- 1. Tambah kolom spesifikasi tambahan pada master spec_templates jika belum ada
ALTER TABLE spec_templates
ADD COLUMN IF NOT EXISTS description TEXT,
ADD COLUMN IF NOT EXISTS composition VARCHAR(255),
ADD COLUMN IF NOT EXISTS care_instruction TEXT;

-- 2. Tambah foreign key spec_template_id pada product_fabrics
ALTER TABLE product_fabrics
ADD COLUMN spec_template_id UUID REFERENCES spec_templates (id) ON DELETE SET NULL,
ALTER COLUMN name
DROP NOT NULL;
-- Name bisa null jika menggunakan name dari SpecTemplate

CREATE INDEX idx_product_fabrics_spec_template_id ON product_fabrics (spec_template_id);