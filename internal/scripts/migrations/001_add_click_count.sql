-- Migration: Add click_count column for batching system
-- This column tracks the number of times each short URL has been clicked

-- Run this migration before starting the application with batching enabled

-- Step 1: Add the column if it doesn't exist
ALTER TABLE urls ADD COLUMN click_count INT DEFAULT 0;

-- Step 2: Create index for better query performance (optional but recommended)
CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);

-- Step 3: Verify the column was added
SELECT column_name, data_type, is_nullable 
FROM information_schema.columns 
WHERE table_name = 'urls' 
ORDER BY ordinal_position;

-- Expected output should include:
-- click_count | integer | YES

-- Step 4: (Optional) Update existing rows to have a baseline count
-- If you want to calculate click counts from logs, do it here
-- For now, they'll all start at 0

-- You can verify the migration worked by checking:
-- SELECT COUNT(*) as rows_with_column FROM urls WHERE click_count IS NOT NULL;
-- Should return the same as: SELECT COUNT(*) FROM urls;
