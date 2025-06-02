-- Create item_images table
CREATE TABLE IF NOT EXISTS item_images (
    id SERIAL PRIMARY KEY,
    item_id INTEGER NOT NULL REFERENCES store_items(id) ON DELETE CASCADE,
    url VARCHAR(500) NOT NULL,
    "order" INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create index for faster lookups
CREATE INDEX idx_item_images_item_id ON item_images(item_id);
CREATE INDEX idx_item_images_order ON item_images(item_id, "order");

-- Migrate existing images data if any
-- This assumes the old images column exists as text[]
DO $$
BEGIN
    IF EXISTS (
        SELECT column_name 
        FROM information_schema.columns 
        WHERE table_name = 'store_items' 
        AND column_name = 'images'
        AND data_type = 'ARRAY'
    ) THEN
        -- Insert existing images into the new table
        INSERT INTO item_images (item_id, url, "order")
        SELECT 
            id,
            unnest(images),
            generate_series(0, array_length(images, 1) - 1)
        FROM store_items
        WHERE images IS NOT NULL AND array_length(images, 1) > 0;
        
        -- Drop the old images column
        ALTER TABLE store_items DROP COLUMN images;
    END IF;
END $$;