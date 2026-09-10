CREATE TABLE materials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    name VARCHAR(255) NOT NULL,
    unit VARCHAR(50) NOT NULL, -- 'meter', 'pcs', 'roll', 'kg'
    unit_price NUMERIC(15, 2) NOT NULL,
    category VARCHAR(100), -- 'kain', 'aksesoris', 'packaging'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_materials_deleted_at ON materials (deleted_at);

CREATE INDEX idx_materials_category ON materials (category);

-- Resep produk (Bill of Materials): produk X butuh material Y sebanyak Z per pcs
CREATE TABLE product_materials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials (id),
    qty_per_unit NUMERIC(15, 4) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (product_id, material_id)
);

CREATE INDEX idx_product_materials_product_id ON product_materials (product_id);

CREATE INDEX idx_product_materials_material_id ON product_materials (material_id);