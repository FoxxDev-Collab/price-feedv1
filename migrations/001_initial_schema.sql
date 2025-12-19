-- Migration 001: Initial Schema
-- Applied by Go app on startup
-- Note: This file is for documentation. The actual migration is embedded in database.go

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Regions table
CREATE TABLE IF NOT EXISTS regions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    state VARCHAR(2) NOT NULL,
    zip_codes TEXT[] NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    username VARCHAR(50) UNIQUE,
    region_id INT REFERENCES regions(id),
    reputation_points INT DEFAULT 0,
    role VARCHAR(20) DEFAULT 'user',
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP
);

-- User sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Stores table
CREATE TABLE IF NOT EXISTS stores (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    street_address VARCHAR(255) NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(2) NOT NULL,
    zip_code VARCHAR(10) NOT NULL,
    region_id INT REFERENCES regions(id),
    store_type VARCHAR(50),
    chain VARCHAR(100),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    verified BOOLEAN DEFAULT FALSE,
    verification_count INT DEFAULT 0,
    created_by INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Items table
CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    brand VARCHAR(100),
    size DECIMAL(10, 3),
    unit VARCHAR(20),
    description TEXT,
    verified BOOLEAN DEFAULT FALSE,
    verification_count INT DEFAULT 0,
    created_by INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Tags table
CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    usage_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Item-Tag junction
CREATE TABLE IF NOT EXISTS item_tags (
    item_id INT REFERENCES items(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    created_by INT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (item_id, tag_id)
);

-- Store prices table
CREATE TABLE IF NOT EXISTS store_prices (
    id SERIAL PRIMARY KEY,
    store_id INT REFERENCES stores(id) ON DELETE CASCADE,
    item_id INT REFERENCES items(id) ON DELETE CASCADE,
    price DECIMAL(10, 2) NOT NULL,
    user_id INT REFERENCES users(id),
    verified_count INT DEFAULT 0,
    last_verified TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Price verifications table
CREATE TABLE IF NOT EXISTS price_verifications (
    id SERIAL PRIMARY KEY,
    price_id INT REFERENCES store_prices(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id),
    is_accurate BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_user_price_verification UNIQUE (price_id, user_id)
);

-- Shopping lists table
CREATE TABLE IF NOT EXISTS shopping_lists (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    target_date DATE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Shopping list items table
CREATE TABLE IF NOT EXISTS shopping_list_items (
    id SERIAL PRIMARY KEY,
    list_id INT REFERENCES shopping_lists(id) ON DELETE CASCADE,
    item_id INT REFERENCES items(id),
    quantity INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Price feed table (activity feed)
CREATE TABLE IF NOT EXISTS price_feed (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    store_id INT REFERENCES stores(id),
    item_id INT REFERENCES items(id),
    price DECIMAL(10, 2),
    action VARCHAR(50),
    region_id INT REFERENCES regions(id),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_stores_region ON stores(region_id);
CREATE INDEX IF NOT EXISTS idx_stores_zip ON stores(zip_code);
CREATE INDEX IF NOT EXISTS idx_store_prices_store ON store_prices(store_id);
CREATE INDEX IF NOT EXISTS idx_store_prices_item ON store_prices(item_id);
CREATE INDEX IF NOT EXISTS idx_store_prices_updated ON store_prices(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_name_trgm ON items USING gin(name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);
CREATE INDEX IF NOT EXISTS idx_tags_usage ON tags(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_price_feed_region ON price_feed(region_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_price_feed_user ON price_feed(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions(expires_at);

-- Insert default region for Colorado Springs
INSERT INTO regions (name, state, zip_codes)
VALUES ('Colorado Springs', 'CO', ARRAY['80901', '80902', '80903', '80904', '80905', '80906', '80907', '80908', '80909', '80910', '80911', '80912', '80913', '80914', '80915', '80916', '80917', '80918', '80919', '80920', '80921', '80922', '80923', '80924', '80925', '80926', '80927', '80928', '80929', '80930', '80931', '80932', '80933', '80934', '80935', '80936', '80937', '80938', '80939', '80941', '80942', '80946', '80947', '80949', '80950', '80951', '80960', '80962', '80970', '80977', '80995', '80997'])
ON CONFLICT DO NOTHING;
