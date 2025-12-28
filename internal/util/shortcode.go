package util

import (
	"crypto/rand"
	"math/big"
)

const (
	// shortCodeLength is the total length of the short code
	shortCodeLength = 8
	// charset for short code characters
	charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// GenerateShortCode generates a unique 8-character alphanumeric short code
// Example: "1A2B3C4D"
func GenerateShortCode() string {
	result := make([]byte, shortCodeLength)

	for i := 0; i < shortCodeLength; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}

	return string(result)
}
