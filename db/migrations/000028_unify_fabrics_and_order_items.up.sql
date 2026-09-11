-- 1. Tambah kolom spesifikasi teknis kain pada tabel materials
ALTER TABLE materials
ADD COLUMN IF NOT EXISTS description TEXT,
ADD COLUMN IF NOT EXISTS composition VARCHAR(255),
ADD COLUMN IF NOT EXISTS care_instruction TEXT,
ADD COLUMN IF NOT EXISTS gsm_info VARCHAR(100);

-- 2. Tambah kolom material_id pada fabric_colors (agar varian warna terhubung langsung ke Master Kain/Material)
ALTER TABLE fabric_colors
ADD COLUMN IF NOT EXISTS material_id UUID REFERENCES materials(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_fabric_colors_material_id ON fabric_colors(material_id);

-- 3. Tambah kolom fabric_id dan qty_per_unit pada product_fabrics
ALTER TABLE product_fabrics
ADD COLUMN IF NOT EXISTS fabric_id UUID REFERENCES materials(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS qty_per_unit NUMERIC(15,4) DEFAULT 1.5000;

CREATE INDEX IF NOT EXISTS idx_product_fabrics_fabric_id ON product_fabrics(fabric_id);

-- 4. Tambah kolom fabric_id dan fabric_color_id pada order_items
ALTER TABLE order_items
ADD COLUMN IF NOT EXISTS fabric_id UUID REFERENCES materials(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS fabric_color_id UUID REFERENCES fabric_colors(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_order_items_fabric_id ON order_items(fabric_id);
CREATE INDEX IF NOT EXISTS idx_order_items_fabric_color_id ON order_items(fabric_color_id);
