-- Migration 004: Add privacy and sharing columns for community data model
-- Applied by Go app on startup

-- Add is_private column to stores table
-- is_private = false: Shared community store (visible to region)
-- is_private = true: Private store (visible only to created_by user)
ALTER TABLE stores ADD COLUMN IF NOT EXISTS is_private BOOLEAN DEFAULT FALSE;

-- Add is_shared column to store_prices table
-- is_shared = true: Price visible to community in region
-- is_shared = false: Price visible only to submitter (user_id)
ALTER TABLE store_prices ADD COLUMN IF NOT EXISTS is_shared BOOLEAN DEFAULT TRUE;

-- Create indexes for efficient filtering
CREATE INDEX IF NOT EXISTS idx_stores_private ON stores(is_private);
CREATE INDEX IF NOT EXISTS idx_stores_created_by ON stores(created_by);
CREATE INDEX IF NOT EXISTS idx_store_prices_shared ON store_prices(is_shared);
CREATE INDEX IF NOT EXISTS idx_store_prices_user ON store_prices(user_id);

-- Composite index for user's private stores query
CREATE INDEX IF NOT EXISTS idx_stores_user_private ON stores(created_by, is_private) WHERE is_private = true;

-- Composite index for community stores in a region
CREATE INDEX IF NOT EXISTS idx_stores_region_public ON stores(region_id, is_private) WHERE is_private = false;
