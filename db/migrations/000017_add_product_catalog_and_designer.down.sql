DROP TABLE IF EXISTS product_model_views;
DROP TABLE IF EXISTS product_models;
DROP TABLE IF EXISTS wholesale_prices;
DROP TABLE IF EXISTS fabric_colors;
DROP TABLE IF EXISTS product_fabrics;

ALTER TABLE products 
  DROP COLUMN IF EXISTS key_features,
  DROP COLUMN IF EXISTS review_count,
  DROP COLUMN IF EXISTS sold_count,
  DROP COLUMN IF EXISTS rating,
  DROP COLUMN IF EXISTS fabric_summary,
  DROP COLUMN IF EXISTS gsm_info,
  DROP COLUMN IF EXISTS slug;