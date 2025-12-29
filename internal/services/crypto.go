package services

import (
	"crypto/sha256"

	"golang.org/x/crypto/pbkdf2"
)

// encryptionSalt for PBKDF2 - must match handlers/auth.go and database/settings_repo.go
var encryptionSalt = []byte("pricefeed-settings-v1")

// DeriveEncryptionKey derives a secure 32-byte key using PBKDF2
// This provides consistent key derivation across all services
func DeriveEncryptionKey(secret string) []byte {
	// Use PBKDF2 with SHA-256 to derive a 32-byte key
	// 100,000 iterations is recommended for PBKDF2-SHA256 as of 2024
	return pbkdf2.Key([]byte(secret), encryptionSalt, 100000, 32, sha256.New)
}
