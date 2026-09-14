CREATE TABLE IF NOT EXISTS spec_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    spec TEXT NOT NULL,
    description TEXT,
    composition VARCHAR(255),
    care_instruction TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE fabric_colors ADD COLUMN IF NOT EXISTS fabric_id UUID;
ALTER TABLE fabric_colors ADD COLUMN IF NOT EXISTS spec_template_id UUID REFERENCES spec_templates(id) ON DELETE CASCADE;
ALTER TABLE fabric_colors ALTER COLUMN material_id DROP NOT NULL;

ALTER TABLE product_fabrics RENAME COLUMN material_id TO fabric_id;
ALTER TABLE product_fabrics ALTER COLUMN fabric_id DROP NOT NULL;
ALTER TABLE product_fabrics ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE product_fabrics ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE product_fabrics ADD COLUMN IF NOT EXISTS composition VARCHAR(255);
ALTER TABLE product_fabrics ADD COLUMN IF NOT EXISTS care_instruction TEXT;
ALTER TABLE product_fabrics ADD COLUMN IF NOT EXISTS base_price DECIMAL(12,2) DEFAULT 0.00;
ALTER TABLE product_fabrics ADD COLUMN IF NOT EXISTS spec_template_id UUID REFERENCES spec_templates(id) ON DELETE SET NULL;
