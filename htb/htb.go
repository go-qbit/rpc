// Package htb implements mitigations for the BREACH attack.
// Heal-the-BREACH involves inserting a random filename into the header section of gzip,
// modifying the overall length of a response, and increasing the time taken to perform BREACH substantially.
//
// Note: this does not remove the possibility, only prolongs it.
package htb

import (
	"crypto/rand"
	"math/big"
)

const (
	paddingSize = 32
	characters  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length      = int64(len(characters))
)

var max = big.NewInt(length)

// RandomString produces a cryptographically secure random string, or panics.
// It will be 32 bytes long, and alphanumeric.
//
// This should be pretty fast, and suitable for concurrent use.
func RandomString() string {
	buf := make([]byte, paddingSize)

	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}

		buf[i] = characters[n.Int64()]
	}

	return string(buf)
}
