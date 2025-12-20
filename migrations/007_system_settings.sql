-- System settings table for storing application configuration
-- This allows settings to be managed through the admin UI instead of environment variables

CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT,
    value_type VARCHAR(20) DEFAULT 'string', -- string, int, bool, json, encrypted
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    description TEXT,
    is_sensitive BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index for category lookups
CREATE INDEX IF NOT EXISTS idx_system_settings_category ON system_settings(category);

-- Insert default SMTP settings
INSERT INTO system_settings (key, value, value_type, category, description, is_sensitive) VALUES
    ('smtp_enabled', 'false', 'bool', 'email', 'Enable SMTP email functionality', false),
    ('smtp_host', '', 'string', 'email', 'SMTP server hostname', false),
    ('smtp_port', '587', 'int', 'email', 'SMTP server port (587 for TLS, 465 for SSL)', false),
    ('smtp_user', '', 'string', 'email', 'SMTP authentication username', false),
    ('smtp_password', '', 'encrypted', 'email', 'SMTP authentication password', true),
    ('smtp_from_addr', 'noreply@pricefeed.app', 'string', 'email', 'From email address', false),
    ('smtp_from_name', 'PriceFeed', 'string', 'email', 'From display name', false)
ON CONFLICT (key) DO NOTHING;

-- Insert default general settings
INSERT INTO system_settings (key, value, value_type, category, description, is_sensitive) VALUES
    ('site_name', 'PriceFeed', 'string', 'general', 'Application name', false),
    ('site_description', 'Community-driven grocery price comparison', 'string', 'general', 'Application description', false),
    ('contact_email', 'support@pricefeed.app', 'string', 'general', 'Contact email address', false),
    ('maintenance_mode', 'false', 'bool', 'general', 'Enable maintenance mode', false)
ON CONFLICT (key) DO NOTHING;

-- Insert default user/auth settings  
INSERT INTO system_settings (key, value, value_type, category, description, is_sensitive) VALUES
    ('allow_registration', 'true', 'bool', 'auth', 'Allow new user registrations', false),
    ('require_email_verify', 'false', 'bool', 'auth', 'Require email verification for new accounts', false),
    ('min_password_length', '8', 'int', 'auth', 'Minimum password length', false),
    ('session_timeout_hours', '24', 'int', 'auth', 'Session timeout in hours', false),
    ('max_login_attempts', '5', 'int', 'auth', 'Maximum failed login attempts before lockout', false),
    ('lockout_duration_minutes', '15', 'int', 'auth', 'Account lockout duration in minutes', false)
ON CONFLICT (key) DO NOTHING;

-- Insert default price settings
INSERT INTO system_settings (key, value, value_type, category, description, is_sensitive) VALUES
    ('price_expiry_days', '7', 'int', 'prices', 'Days before prices are considered stale', false),
    ('verification_threshold', '3', 'int', 'prices', 'Verifications needed to mark price as verified', false),
    ('allow_anonymous_prices', 'true', 'bool', 'prices', 'Allow price viewing without login', false),
    ('require_receipt', 'false', 'bool', 'prices', 'Require receipt photo for price submissions', false),
    ('max_price_deviation', '50', 'int', 'prices', 'Flag prices deviating more than this % from average', false)
ON CONFLICT (key) DO NOTHING;

-- Insert default reputation settings
INSERT INTO system_settings (key, value, value_type, category, description, is_sensitive) VALUES
    ('points_price_submission', '5', 'int', 'reputation', 'Points for submitting a price', false),
    ('points_verification', '2', 'int', 'reputation', 'Points for verifying a price', false),
    ('points_store_added', '10', 'int', 'reputation', 'Points for adding a store', false),
    ('points_item_added', '3', 'int', 'reputation', 'Points for adding an item', false),
    ('level_bronze', '100', 'int', 'reputation', 'Points needed for Bronze level', false),
    ('level_silver', '500', 'int', 'reputation', 'Points needed for Silver level', false),
    ('level_gold', '1000', 'int', 'reputation', 'Points needed for Gold level', false),
    ('level_platinum', '5000', 'int', 'reputation', 'Points needed for Platinum level', false)
ON CONFLICT (key) DO NOTHING;

-- Insert default API settings
INSERT INTO system_settings (key, value, value_type, category, description, is_sensitive) VALUES
    ('api_rate_limit', '60', 'int', 'api', 'API rate limit (requests per minute)', false),
    ('cors_origins', '*', 'string', 'api', 'CORS allowed origins', false),
    ('enable_public_api', 'true', 'bool', 'api', 'Enable public API access', false),
    ('require_api_key', 'true', 'bool', 'api', 'Require API key for external access', false)
ON CONFLICT (key) DO NOTHING;

-- Function to update timestamp
CREATE OR REPLACE FUNCTION update_system_settings_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update timestamp
DROP TRIGGER IF EXISTS trigger_system_settings_updated ON system_settings;
CREATE TRIGGER trigger_system_settings_updated
    BEFORE UPDATE ON system_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_system_settings_timestamp();
