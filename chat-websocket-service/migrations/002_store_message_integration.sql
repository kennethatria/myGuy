-- Migration to integrate store messages into unified messages table
BEGIN;

-- 1. Add store_item_id to messages table
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS store_item_id INTEGER;

-- 2. Create indexes for store messages
CREATE INDEX IF NOT EXISTS idx_messages_store_item ON messages(store_item_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_store_item_participants ON messages(store_item_id, sender_id, recipient_id);

-- 3. Add foreign key if store_items table exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'store_items'
    ) THEN
        -- Drop existing constraint if it exists
        IF EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_messages_store_item'
        ) THEN
            ALTER TABLE messages DROP CONSTRAINT fk_messages_store_item;
        END IF;

        -- Add foreign key with cascade delete
        ALTER TABLE messages 
        ADD CONSTRAINT fk_messages_store_item 
        FOREIGN KEY (store_item_id) 
        REFERENCES store_items(id)
        ON DELETE CASCADE;
    END IF;
END
$$;

-- 4. Migrate existing store messages if the table exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'store_messages'
    ) THEN
        -- Migrate data
        INSERT INTO messages (
            store_item_id,
            sender_id,
            recipient_id,
            content,
            original_content,
            created_at,
            is_read,
            read_at
        )
        SELECT 
            store_item_id,
            sender_id,
            recipient_id,
            content,
            original_content,
            created_at,
            is_read,
            read_at
        FROM store_messages
        ON CONFLICT DO NOTHING;

        -- Drop old table
        DROP TABLE store_messages;
    END IF;
END
$$;

-- 5. Update message counts in store_items table if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'store_items'
    ) AND 
    EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'store_items' 
        AND column_name = 'message_count'
    ) THEN
        -- Update message counts
        UPDATE store_items si
        SET message_count = (
            SELECT COUNT(*)
            FROM messages m
            WHERE m.store_item_id = si.id
        );

        -- Create or replace message count update function
        CREATE OR REPLACE FUNCTION update_store_item_message_count()
        RETURNS TRIGGER AS $$
        BEGIN
            IF (TG_OP = 'INSERT' OR TG_OP = 'DELETE') THEN
                UPDATE store_items
                SET message_count = (
                    SELECT COUNT(*)
                    FROM messages
                    WHERE store_item_id = COALESCE(NEW.store_item_id, OLD.store_item_id)
                )
                WHERE id = COALESCE(NEW.store_item_id, OLD.store_item_id);
            END IF;
            RETURN NULL;
        END;
        $$ LANGUAGE plpgsql;

        -- Create trigger if it doesn't exist
        DROP TRIGGER IF EXISTS store_message_count_trigger ON messages;
        CREATE TRIGGER store_message_count_trigger
        AFTER INSERT OR DELETE ON messages
        FOR EACH ROW
        WHEN (NEW.store_item_id IS NOT NULL OR OLD.store_item_id IS NOT NULL)
        EXECUTE FUNCTION update_store_item_message_count();
    END IF;
END
$$;

COMMIT;
