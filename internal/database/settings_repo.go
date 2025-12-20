package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
)

// SystemSetting represents a configuration setting stored in the database
type SystemSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ValueType   string    `json:"value_type"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	IsSensitive bool      `json:"is_sensitive"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SettingsMap is a map of setting keys to values
type SettingsMap map[string]interface{}

var ErrSettingNotFound = errors.New("setting not found")

// encryptionKey should be derived from JWT secret or a dedicated settings key
// For now we'll use a simple approach - in production, use proper key derivation
func getEncryptionKey(secret string) []byte {
	// Pad or truncate to 32 bytes for AES-256
	key := make([]byte, 32)
	copy(key, []byte(secret))
	return key
}

// encrypt encrypts a string value
func encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts an encrypted string value
func decrypt(ciphertext string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := string(data[:nonceSize]), string(data[nonceSize:])
	plaintext, err := gcm.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GetSetting retrieves a single setting by key
func (db *DB) GetSetting(ctx context.Context, key string, encryptionKey []byte) (*SystemSetting, error) {
	var s SystemSetting
	err := db.Pool.QueryRow(ctx, `
		SELECT key, value, value_type, category, description, is_sensitive, created_at, updated_at
		FROM system_settings
		WHERE key = $1
	`, key).Scan(&s.Key, &s.Value, &s.ValueType, &s.Category, &s.Description, &s.IsSensitive, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, ErrSettingNotFound
		}
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	// Decrypt if encrypted
	if s.ValueType == "encrypted" && s.Value != "" && encryptionKey != nil {
		decrypted, err := decrypt(s.Value, encryptionKey)
		if err == nil {
			s.Value = decrypted
		}
		// If decryption fails, return empty (might be unencrypted old value)
	}

	return &s, nil
}

// GetSettingValue retrieves just the value of a setting, converted to the appropriate type
func (db *DB) GetSettingValue(ctx context.Context, key string, encryptionKey []byte) (interface{}, error) {
	setting, err := db.GetSetting(ctx, key, encryptionKey)
	if err != nil {
		return nil, err
	}

	return convertSettingValue(setting.Value, setting.ValueType), nil
}

// GetSettingString retrieves a setting as a string
func (db *DB) GetSettingString(ctx context.Context, key string, defaultValue string, encryptionKey []byte) string {
	setting, err := db.GetSetting(ctx, key, encryptionKey)
	if err != nil {
		return defaultValue
	}
	if setting.Value == "" {
		return defaultValue
	}
	return setting.Value
}

// GetSettingInt retrieves a setting as an integer
func (db *DB) GetSettingInt(ctx context.Context, key string, defaultValue int, encryptionKey []byte) int {
	setting, err := db.GetSetting(ctx, key, encryptionKey)
	if err != nil {
		return defaultValue
	}
	val, err := strconv.Atoi(setting.Value)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetSettingBool retrieves a setting as a boolean
func (db *DB) GetSettingBool(ctx context.Context, key string, defaultValue bool, encryptionKey []byte) bool {
	setting, err := db.GetSetting(ctx, key, encryptionKey)
	if err != nil {
		return defaultValue
	}
	val, err := strconv.ParseBool(setting.Value)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetSettingsByCategory retrieves all settings in a category
func (db *DB) GetSettingsByCategory(ctx context.Context, category string, encryptionKey []byte) ([]SystemSetting, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT key, value, value_type, category, description, is_sensitive, created_at, updated_at
		FROM system_settings
		WHERE category = $1
		ORDER BY key
	`, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings by category: %w", err)
	}
	defer rows.Close()

	var settings []SystemSetting
	for rows.Next() {
		var s SystemSetting
		if err := rows.Scan(&s.Key, &s.Value, &s.ValueType, &s.Category, &s.Description, &s.IsSensitive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}

		// Decrypt if encrypted
		if s.ValueType == "encrypted" && s.Value != "" && encryptionKey != nil {
			decrypted, err := decrypt(s.Value, encryptionKey)
			if err == nil {
				s.Value = decrypted
			}
		}

		// Mask sensitive values for output (show only if explicitly requested)
		if s.IsSensitive && s.Value != "" {
			s.Value = "••••••••"
		}

		settings = append(settings, s)
	}

	return settings, nil
}

// GetSettingsByCategoryAsMap retrieves all settings in a category as a map
func (db *DB) GetSettingsByCategoryAsMap(ctx context.Context, category string, encryptionKey []byte, includeSensitive bool) (SettingsMap, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT key, value, value_type, is_sensitive
		FROM system_settings
		WHERE category = $1
	`, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings by category: %w", err)
	}
	defer rows.Close()

	result := make(SettingsMap)
	for rows.Next() {
		var key, value, valueType string
		var isSensitive bool
		if err := rows.Scan(&key, &value, &valueType, &isSensitive); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}

		// Decrypt if encrypted
		if valueType == "encrypted" && value != "" && encryptionKey != nil {
			decrypted, err := decrypt(value, encryptionKey)
			if err == nil {
				value = decrypted
			}
		}

		// Mask sensitive values unless explicitly requested
		if isSensitive && !includeSensitive && value != "" {
			result[key] = "••••••••"
		} else {
			result[key] = convertSettingValue(value, valueType)
		}
	}

	return result, nil
}

// GetAllSettings retrieves all settings
func (db *DB) GetAllSettings(ctx context.Context, encryptionKey []byte) (map[string][]SystemSetting, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT key, value, value_type, category, description, is_sensitive, created_at, updated_at
		FROM system_settings
		ORDER BY category, key
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all settings: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]SystemSetting)
	for rows.Next() {
		var s SystemSetting
		if err := rows.Scan(&s.Key, &s.Value, &s.ValueType, &s.Category, &s.Description, &s.IsSensitive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}

		// Mask sensitive values
		if s.IsSensitive && s.Value != "" {
			s.Value = "••••••••"
		}

		result[s.Category] = append(result[s.Category], s)
	}

	return result, nil
}

// SetSetting updates or creates a setting
func (db *DB) SetSetting(ctx context.Context, key, value string, encryptionKey []byte) error {
	// First, get the existing setting to check if it should be encrypted
	var valueType string
	err := db.Pool.QueryRow(ctx, `SELECT value_type FROM system_settings WHERE key = $1`, key).Scan(&valueType)
	if err != nil {
		// Setting doesn't exist, just insert without encryption
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO system_settings (key, value, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
		`, key, value)
		return err
	}

	// Encrypt if needed
	finalValue := value
	if valueType == "encrypted" && value != "" && value != "••••••••" && encryptionKey != nil {
		encrypted, err := encrypt(value, encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt value: %w", err)
		}
		finalValue = encrypted
	}

	// Don't update if masked value is submitted
	if value == "••••••••" {
		return nil
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE system_settings SET value = $2, updated_at = NOW() WHERE key = $1
	`, key, finalValue)

	return err
}

// SetSettings updates multiple settings at once
func (db *DB) SetSettings(ctx context.Context, settings map[string]string, encryptionKey []byte) error {
	for key, value := range settings {
		if err := db.SetSetting(ctx, key, value, encryptionKey); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}
	return nil
}

// SetSettingWithMeta creates or updates a setting with full metadata
func (db *DB) SetSettingWithMeta(ctx context.Context, setting SystemSetting, encryptionKey []byte) error {
	// Encrypt if needed
	finalValue := setting.Value
	if setting.ValueType == "encrypted" && setting.Value != "" && encryptionKey != nil {
		encrypted, err := encrypt(setting.Value, encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt value: %w", err)
		}
		finalValue = encrypted
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO system_settings (key, value, value_type, category, description, is_sensitive)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (key) DO UPDATE SET
			value = $2,
			value_type = $3,
			category = $4,
			description = $5,
			is_sensitive = $6,
			updated_at = NOW()
	`, setting.Key, finalValue, setting.ValueType, setting.Category, setting.Description, setting.IsSensitive)

	return err
}

// DeleteSetting removes a setting
func (db *DB) DeleteSetting(ctx context.Context, key string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM system_settings WHERE key = $1`, key)
	return err
}

// Helper function to convert setting value to appropriate type
func convertSettingValue(value, valueType string) interface{} {
	switch valueType {
	case "int":
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
		return 0
	case "bool":
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
		return false
	case "json":
		var v interface{}
		if err := json.Unmarshal([]byte(value), &v); err == nil {
			return v
		}
		return nil
	default:
		return value
	}
}

// SMTPConfig holds SMTP configuration from database
type SMTPConfig struct {
	Enabled  bool
	Host     string
	Port     int
	User     string
	Password string
	FromAddr string
	FromName string
}

// GetSMTPConfig retrieves all SMTP settings as a config struct
func (db *DB) GetSMTPConfig(ctx context.Context, encryptionKey []byte) (*SMTPConfig, error) {
	config := &SMTPConfig{
		Port:     587,
		FromAddr: "noreply@pricefeed.app",
		FromName: "PriceFeed",
	}

	config.Enabled = db.GetSettingBool(ctx, "smtp_enabled", false, encryptionKey)
	config.Host = db.GetSettingString(ctx, "smtp_host", "", encryptionKey)
	config.Port = db.GetSettingInt(ctx, "smtp_port", 587, encryptionKey)
	config.User = db.GetSettingString(ctx, "smtp_user", "", encryptionKey)
	config.FromAddr = db.GetSettingString(ctx, "smtp_from_addr", "noreply@pricefeed.app", encryptionKey)
	config.FromName = db.GetSettingString(ctx, "smtp_from_name", "PriceFeed", encryptionKey)

	// Get password (need to decrypt)
	setting, err := db.GetSetting(ctx, "smtp_password", encryptionKey)
	if err == nil && setting.Value != "" {
		config.Password = setting.Value
	}

	return config, nil
}
