-- Migration 008: Add privacy column to items table for user data isolation
-- Applied by Go app on startup

-- Add is_private column to items table
-- is_private = true (default): Item visible only to creator
-- is_private = false: Shared community item (visible to all)
-- NOTE: Default TRUE for items because items are personal to each user's tracking
ALTER TABLE items ADD COLUMN IF NOT EXISTS is_private BOOLEAN DEFAULT TRUE;

-- Create indexes for efficient filtering
CREATE INDEX IF NOT EXISTS idx_items_private ON items(is_private);
CREATE INDEX IF NOT EXISTS idx_items_created_by ON items(created_by);

-- Composite index for user's items query (their own items)
CREATE INDEX IF NOT EXISTS idx_items_user_private ON items(created_by, is_private) WHERE is_private = true;

-- Update existing items to be private (owned by their creator)
-- This ensures existing user data remains private
UPDATE items SET is_private = true WHERE is_private IS NULL;
