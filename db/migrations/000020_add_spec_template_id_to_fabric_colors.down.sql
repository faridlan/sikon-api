-- Rollback migration ke-20
DROP INDEX IF EXISTS idx_fabric_colors_spec_template_id;

-- Hapus data warna yang tidak punya fabric_id sebelum mengembalikan constraint NOT NULL
DELETE FROM fabric_colors WHERE fabric_id IS NULL;

ALTER TABLE fabric_colors ALTER COLUMN fabric_id SET NOT NULL;

ALTER TABLE fabric_colors DROP COLUMN IF EXISTS spec_template_id;