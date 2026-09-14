-- Material sudah punya Description/Composition/CareInstruction/GSMInfo/Colors sejak migration 000028.
-- Migration ini membuang jalur redundan yang masih nyantol: spec_templates, dan field-field
-- duplikat di product_fabrics/fabric_colors yang seharusnya cukup diambil dari Material.

-- 1. Sederhanakan product_fabrics: buang semua field deskriptif (sekarang diambil dari Material
--    yang di-link), rename fabric_id -> material_id biar konsisten sama konsep barunya, wajib diisi.
ALTER TABLE product_fabrics DROP COLUMN IF EXISTS name;
ALTER TABLE product_fabrics DROP COLUMN IF EXISTS description;
ALTER TABLE product_fabrics DROP COLUMN IF EXISTS composition;
ALTER TABLE product_fabrics DROP COLUMN IF EXISTS care_instruction;
ALTER TABLE product_fabrics DROP COLUMN IF EXISTS base_price;
ALTER TABLE product_fabrics DROP COLUMN IF EXISTS spec_template_id;

DELETE FROM product_fabrics WHERE fabric_id IS NULL; -- baris lama tanpa link material, tidak valid lagi
ALTER TABLE product_fabrics RENAME COLUMN fabric_id TO material_id;
ALTER TABLE product_fabrics ALTER COLUMN material_id SET NOT NULL;

-- 2. Sederhanakan fabric_colors: warna sekarang murni milik Material, bukan lagi milik
--    ProductFabric atau SpecTemplate.
DELETE FROM fabric_colors WHERE material_id IS NULL; -- baris lama tanpa link material, tidak valid lagi
ALTER TABLE fabric_colors DROP COLUMN IF EXISTS fabric_id;
ALTER TABLE fabric_colors DROP COLUMN IF EXISTS spec_template_id;
ALTER TABLE fabric_colors ALTER COLUMN material_id SET NOT NULL;

-- 3. wholesale_prices.fabric_id tetap merujuk ke product_fabrics(id) - tidak berubah,
--    PK product_fabrics tidak berubah, cuma kolomnya yang berkurang.

-- 4. Hapus spec_templates sepenuhnya.
DROP TABLE IF EXISTS spec_templates CASCADE;
