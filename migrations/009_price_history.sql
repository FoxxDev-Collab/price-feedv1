-- Migration: Price History Tracking
-- Records historical price changes for items at stores

CREATE TABLE IF NOT EXISTS price_history (
    id SERIAL PRIMARY KEY,
    store_id INT REFERENCES stores(id) ON DELETE CASCADE,
    item_id INT REFERENCES items(id) ON DELETE CASCADE,
    price DECIMAL(10, 2) NOT NULL,
    previous_price DECIMAL(10, 2),
    user_id INT REFERENCES users(id) ON DELETE SET NULL,
    recorded_at TIMESTAMP DEFAULT NOW()
);

-- Index for querying history by item and store (most common query)
CREATE INDEX IF NOT EXISTS idx_price_history_item_store ON price_history(item_id, store_id, recorded_at DESC);

-- Index for querying all history for an item across stores
CREATE INDEX IF NOT EXISTS idx_price_history_item ON price_history(item_id, recorded_at DESC);

-- Index for querying by store
CREATE INDEX IF NOT EXISTS idx_price_history_store ON price_history(store_id, recorded_at DESC);
