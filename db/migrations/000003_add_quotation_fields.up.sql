ALTER TABLE orders
ADD COLUMN valid_until DATE NULL,
ADD COLUMN terms_conditions TEXT NULL;