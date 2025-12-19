-- Migration 002: Add store plans and optimization tables
-- Applied by Go app on startup

-- Store plans table (generated optimizations for shopping lists)
CREATE TABLE IF NOT EXISTS store_plans (
    id SERIAL PRIMARY KEY,
    list_id INT REFERENCES shopping_lists(id) ON DELETE CASCADE,
    total_savings DECIMAL(10, 2),
    recommended_strategy TEXT,
    generated_at TIMESTAMP DEFAULT NOW()
);

-- Store plan items table (items assigned to specific stores)
CREATE TABLE IF NOT EXISTS store_plan_items (
    id SERIAL PRIMARY KEY,
    plan_id INT REFERENCES store_plans(id) ON DELETE CASCADE,
    store_id INT REFERENCES stores(id),
    item_id INT REFERENCES items(id),
    quantity INT NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Add unique constraint on store addresses to prevent duplicates
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'unique_store_address'
    ) THEN
        ALTER TABLE stores ADD CONSTRAINT unique_store_address
        UNIQUE (street_address, state, zip_code, region_id);
    END IF;
END $$;

-- Create address normalization function
CREATE OR REPLACE FUNCTION normalize_address(addr TEXT) RETURNS TEXT AS $$
BEGIN
    RETURN LOWER(
        REGEXP_REPLACE(
            REGEXP_REPLACE(
                REGEXP_REPLACE(
                    REGEXP_REPLACE(
                        REGEXP_REPLACE(
                            REGEXP_REPLACE(
                                REGEXP_REPLACE(
                                    REGEXP_REPLACE(addr, '\bstreet\b', 'st', 'gi'),
                                    '\bavenue\b', 'ave', 'gi'),
                                '\bboulevard\b', 'blvd', 'gi'),
                            '\bdrive\b', 'dr', 'gi'),
                        '\broad\b', 'rd', 'gi'),
                    '\bparkway\b', 'pkwy', 'gi'),
                '\bcircle\b', 'cir', 'gi'),
            '\bcourt\b', 'ct', 'gi')
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Create indexes for new tables
CREATE INDEX IF NOT EXISTS idx_store_plans_list ON store_plans(list_id);
CREATE INDEX IF NOT EXISTS idx_store_plan_items_plan ON store_plan_items(plan_id);
CREATE INDEX IF NOT EXISTS idx_store_plan_items_store ON store_plan_items(store_id);
