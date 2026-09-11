DROP INDEX IF EXISTS idx_order_items_fabric_color_id;
DROP INDEX IF EXISTS idx_order_items_fabric_id;
ALTER TABLE order_items DROP COLUMN IF EXISTS fabric_color_id;
ALTER TABLE order_items DROP COLUMN IF EXISTS fabric_id;

DROP INDEX IF EXISTS idx_product_fabrics_fabric_id;
ALTER TABLE product_fabrics DROP COLUMN IF EXISTS qty_per_unit;
ALTER TABLE product_fabrics DROP COLUMN IF EXISTS fabric_id;

DROP INDEX IF EXISTS idx_fabric_colors_material_id;
ALTER TABLE fabric_colors DROP COLUMN IF EXISTS material_id;

ALTER TABLE materials DROP COLUMN IF EXISTS gsm_info;
ALTER TABLE materials DROP COLUMN IF EXISTS care_instruction;
ALTER TABLE materials DROP COLUMN IF EXISTS composition;
ALTER TABLE materials DROP COLUMN IF EXISTS description;
