-- 1. Tambahkan kolom pendukung di tabel products
ALTER TABLE products 
  ADD COLUMN IF NOT EXISTS slug VARCHAR(255) UNIQUE,
  ADD COLUMN IF NOT EXISTS gsm_info VARCHAR(100),
  ADD COLUMN IF NOT EXISTS fabric_summary VARCHAR(100),
  ADD COLUMN IF NOT EXISTS rating DECIMAL(3,2) DEFAULT 0.00,
  ADD COLUMN IF NOT EXISTS sold_count INT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS review_count INT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS key_features JSONB DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_products_slug ON products (slug);

-- 2. Buat tabel product_fabrics (Opsi Bahan Kain + Custom Base Price per Kain)
CREATE TABLE IF NOT EXISTS product_fabrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL, -- e.g. "Nagata Drill", "American Drill"
    description TEXT,
    composition VARCHAR(255),
    care_instruction TEXT,
    base_price DECIMAL(12,2) DEFAULT 0.00, -- e.g. 395000 (Nagata) vs 375000 (American)
    price_adjustment DECIMAL(12,2) DEFAULT 0.00, -- Opsi penyesuaian relatif
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_product_fabrics_product_id ON product_fabrics(product_id);

-- 3. Buat tabel fabric_colors (Warna Khusus Per Bahan Kain)
CREATE TABLE IF NOT EXISTS fabric_colors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fabric_id UUID NOT NULL REFERENCES product_fabrics(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL, -- e.g. "Khaki", "Navy", "Olive"
    hex_code VARCHAR(20) NOT NULL, -- e.g. "#C2B280"
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_fabric_colors_fabric_id ON fabric_colors(fabric_id);

-- 4. Buat tabel wholesale_prices (Harga Grosir Fleksibel: Produk / Kain)
CREATE TABLE IF NOT EXISTS wholesale_prices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    fabric_id UUID REFERENCES product_fabrics(id) ON DELETE CASCADE, -- NULLABLE!
    min_qty INT NOT NULL,
    max_qty INT, -- Boleh NULL jika min_qty+ (e.g. 6 pcs ke atas)
    unit_price DECIMAL(12,2) NOT NULL, -- e.g. 375000 (Nagata min 6)
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_wholesale_prices_product_id ON wholesale_prices(product_id);
CREATE INDEX IF NOT EXISTS idx_wholesale_prices_fabric_id ON wholesale_prices(fabric_id);

-- 5. Buat tabel product_models (Template Canvas Designer)
CREATE TABLE IF NOT EXISTS product_models (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL, -- e.g. 'long_sleeve', 'short_sleeve'
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_product_models_product_id ON product_models(product_id);

-- 6. Buat tabel product_model_views (Aset PNG Art & Mask per Sisi)
CREATE TABLE IF NOT EXISTS product_model_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_model_id UUID NOT NULL REFERENCES product_models(id) ON DELETE CASCADE,
    side VARCHAR(20) NOT NULL, -- 'front' atau 'back'
    art_url VARCHAR(255) NOT NULL,
    mask_url VARCHAR(255) NOT NULL,
    width INT DEFAULT 600,
    height INT DEFAULT 600,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_product_model_views_model_id ON product_model_views(product_model_id);