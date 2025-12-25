package state

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// IDLength is the full length of an environment ID in hex characters.
const IDLength = 32

// ShortIDLength is the display length of an environment ID.
const ShortIDLength = 12

// GenerateID generates a new 32-character hex ID using crypto/rand.
func GenerateID() (string, error) {
	b := make([]byte, IDLength/2) // 16 bytes = 128 bits
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ShortID returns the first 12 characters of an ID for display.
func ShortID(id string) string {
	if len(id) < ShortIDLength {
		return id
	}
	return id[:ShortIDLength]
}
