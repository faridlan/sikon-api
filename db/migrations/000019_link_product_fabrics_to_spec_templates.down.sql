DROP INDEX IF EXISTS idx_product_fabrics_spec_template_id;

ALTER TABLE product_fabrics DROP COLUMN IF EXISTS spec_template_id;

ALTER TABLE spec_templates
DROP COLUMN IF EXISTS description,
DROP COLUMN IF EXISTS composition,
DROP COLUMN IF EXISTS care_instruction;