package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/foxxcyber/price-feed/internal/config"
)

// DB wraps the connection pool
type DB struct {
	Pool *pgxpool.Pool
}

// Connect creates a new database connection pool
func Connect(databaseURL string) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	// Configure pool
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("Database connected successfully")
	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.Pool.Close()
}

// RunMigrations runs all database migrations
func RunMigrations(db *DB) error {
	ctx := context.Background()

	// Create migrations table if it doesn't exist
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Run each migration
	for version, migration := range migrations {
		// Check if migration already applied
		var exists bool
		err := db.Pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration %d: %w", version, err)
		}

		if exists {
			continue
		}

		// Apply migration
		log.Printf("Applying migration %d...", version)
		_, err = db.Pool.Exec(ctx, migration)
		if err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", version, err)
		}

		// Record migration
		_, err = db.Pool.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)",
			version,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %w", version, err)
		}

		log.Printf("Migration %d applied successfully", version)
	}

	return nil
}

// EnsureAdminUser creates the admin user if it doesn't exist
func EnsureAdminUser(db *DB, cfg *config.Config) error {
	if cfg.AdminPassword == "" {
		log.Println("ADMIN_PASSWORD not set, skipping admin user creation")
		return nil
	}

	ctx := context.Background()

	// Check if admin exists
	var exists bool
	err := db.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)",
		cfg.AdminEmail,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check for admin user: %w", err)
	}

	if exists {
		log.Println("Admin user already exists")
		return nil
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// Create admin user
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO users (email, password_hash, username, role, email_verified)
		VALUES ($1, $2, 'admin', 'admin', true)
	`, cfg.AdminEmail, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Printf("Admin user created: %s", cfg.AdminEmail)
	return nil
}

// migrations is an ordered map of migration version to SQL
var migrations = map[int]string{
	1: migration001,
	2: migration002,
	3: migration003,
	4: migration004,
	5: migration005,
	6: migration006,
}

const migration001 = `
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
`

const migration002 = `
-- Migration 002: Add store plans and optimization tables

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
`

const migration003 = `
-- Migration 003: Add all US states as regions
-- Cities will be added in subsequent migrations

INSERT INTO regions (name, state, zip_codes) VALUES
    ('Alabama', 'AL', ARRAY[]::TEXT[]),
    ('Alaska', 'AK', ARRAY[]::TEXT[]),
    ('Arizona', 'AZ', ARRAY[]::TEXT[]),
    ('Arkansas', 'AR', ARRAY[]::TEXT[]),
    ('California', 'CA', ARRAY[]::TEXT[]),
    ('Colorado', 'CO', ARRAY[]::TEXT[]),
    ('Connecticut', 'CT', ARRAY[]::TEXT[]),
    ('Delaware', 'DE', ARRAY[]::TEXT[]),
    ('Florida', 'FL', ARRAY[]::TEXT[]),
    ('Georgia', 'GA', ARRAY[]::TEXT[]),
    ('Hawaii', 'HI', ARRAY[]::TEXT[]),
    ('Idaho', 'ID', ARRAY[]::TEXT[]),
    ('Illinois', 'IL', ARRAY[]::TEXT[]),
    ('Indiana', 'IN', ARRAY[]::TEXT[]),
    ('Iowa', 'IA', ARRAY[]::TEXT[]),
    ('Kansas', 'KS', ARRAY[]::TEXT[]),
    ('Kentucky', 'KY', ARRAY[]::TEXT[]),
    ('Louisiana', 'LA', ARRAY[]::TEXT[]),
    ('Maine', 'ME', ARRAY[]::TEXT[]),
    ('Maryland', 'MD', ARRAY[]::TEXT[]),
    ('Massachusetts', 'MA', ARRAY[]::TEXT[]),
    ('Michigan', 'MI', ARRAY[]::TEXT[]),
    ('Minnesota', 'MN', ARRAY[]::TEXT[]),
    ('Mississippi', 'MS', ARRAY[]::TEXT[]),
    ('Missouri', 'MO', ARRAY[]::TEXT[]),
    ('Montana', 'MT', ARRAY[]::TEXT[]),
    ('Nebraska', 'NE', ARRAY[]::TEXT[]),
    ('Nevada', 'NV', ARRAY[]::TEXT[]),
    ('New Hampshire', 'NH', ARRAY[]::TEXT[]),
    ('New Jersey', 'NJ', ARRAY[]::TEXT[]),
    ('New Mexico', 'NM', ARRAY[]::TEXT[]),
    ('New York', 'NY', ARRAY[]::TEXT[]),
    ('North Carolina', 'NC', ARRAY[]::TEXT[]),
    ('North Dakota', 'ND', ARRAY[]::TEXT[]),
    ('Ohio', 'OH', ARRAY[]::TEXT[]),
    ('Oklahoma', 'OK', ARRAY[]::TEXT[]),
    ('Oregon', 'OR', ARRAY[]::TEXT[]),
    ('Pennsylvania', 'PA', ARRAY[]::TEXT[]),
    ('Rhode Island', 'RI', ARRAY[]::TEXT[]),
    ('South Carolina', 'SC', ARRAY[]::TEXT[]),
    ('South Dakota', 'SD', ARRAY[]::TEXT[]),
    ('Tennessee', 'TN', ARRAY[]::TEXT[]),
    ('Texas', 'TX', ARRAY[]::TEXT[]),
    ('Utah', 'UT', ARRAY[]::TEXT[]),
    ('Vermont', 'VT', ARRAY[]::TEXT[]),
    ('Virginia', 'VA', ARRAY[]::TEXT[]),
    ('Washington', 'WA', ARRAY[]::TEXT[]),
    ('West Virginia', 'WV', ARRAY[]::TEXT[]),
    ('Wisconsin', 'WI', ARRAY[]::TEXT[]),
    ('Wyoming', 'WY', ARRAY[]::TEXT[]),
    -- US Territories
    ('District of Columbia', 'DC', ARRAY[]::TEXT[]),
    ('Puerto Rico', 'PR', ARRAY[]::TEXT[]),
    ('Guam', 'GU', ARRAY[]::TEXT[]),
    ('US Virgin Islands', 'VI', ARRAY[]::TEXT[]),
    ('American Samoa', 'AS', ARRAY[]::TEXT[]),
    ('Northern Mariana Islands', 'MP', ARRAY[]::TEXT[])
ON CONFLICT DO NOTHING;
`

const migration004 = `
-- Migration 004: Add privacy and sharing columns for community data model

-- Add is_private column to stores table
-- is_private = false: Shared community store (visible to region)
-- is_private = true: Private store (visible only to created_by user)
ALTER TABLE stores ADD COLUMN IF NOT EXISTS is_private BOOLEAN DEFAULT FALSE;

-- Add is_shared column to store_prices table
-- is_shared = true: Price visible to community in region
-- is_shared = false: Price visible only to submitter (user_id)
ALTER TABLE store_prices ADD COLUMN IF NOT EXISTS is_shared BOOLEAN DEFAULT TRUE;

-- Add unique constraint on shopping_list_items for upsert support
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'unique_list_item'
    ) THEN
        ALTER TABLE shopping_list_items ADD CONSTRAINT unique_list_item
        UNIQUE (list_id, item_id);
    END IF;
END $$;

-- Create indexes for efficient filtering
CREATE INDEX IF NOT EXISTS idx_stores_private ON stores(is_private);
CREATE INDEX IF NOT EXISTS idx_stores_created_by ON stores(created_by);
CREATE INDEX IF NOT EXISTS idx_store_prices_shared ON store_prices(is_shared);
CREATE INDEX IF NOT EXISTS idx_store_prices_user ON store_prices(user_id);

-- Composite index for user's private stores query
CREATE INDEX IF NOT EXISTS idx_stores_user_private ON stores(created_by, is_private) WHERE is_private = true;

-- Composite index for community stores in a region
CREATE INDEX IF NOT EXISTS idx_stores_region_public ON stores(region_id, is_private) WHERE is_private = false;

-- Index for shopping lists by user
CREATE INDEX IF NOT EXISTS idx_shopping_lists_user ON shopping_lists(user_id);
CREATE INDEX IF NOT EXISTS idx_shopping_list_items_list ON shopping_list_items(list_id);
`

const migration005 = `
-- Migration 005: Add status and completed_at columns to shopping_lists for list archiving

-- Add status column (active, completed)
ALTER TABLE shopping_lists ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';

-- Add completed_at timestamp for when list was marked as complete
ALTER TABLE shopping_lists ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP;

-- Create index for filtering by status
CREATE INDEX IF NOT EXISTS idx_shopping_lists_status ON shopping_lists(status);

-- Composite index for user's active lists (most common query)
CREATE INDEX IF NOT EXISTS idx_shopping_lists_user_status ON shopping_lists(user_id, status);
`

const migration006 = `
-- Migration 006: Add location fields to users table for Google Maps integration

ALTER TABLE users ADD COLUMN IF NOT EXISTS street_address VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS city VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS state VARCHAR(2);
ALTER TABLE users ADD COLUMN IF NOT EXISTS zip_code VARCHAR(10);
ALTER TABLE users ADD COLUMN IF NOT EXISTS latitude DECIMAL(10, 8);
ALTER TABLE users ADD COLUMN IF NOT EXISTS longitude DECIMAL(11, 8);
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_place_id VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_users_location ON users(latitude, longitude) WHERE latitude IS NOT NULL;
`
