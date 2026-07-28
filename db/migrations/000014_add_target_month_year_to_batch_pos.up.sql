ALTER TABLE batch_pos 
ADD COLUMN target_month INT,
ADD COLUMN target_year INT;

-- Update data lama agar tidak null (mengambil dari start_date)
UPDATE batch_pos 
SET target_month = EXTRACT(MONTH FROM start_date),
    target_year = EXTRACT(YEAR FROM start_date)
WHERE target_month IS NULL;