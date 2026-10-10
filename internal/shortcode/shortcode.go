// Package shortcode is the single source of truth for short-code
// generation and validation.
//
// Both repository (generation) and service (validation) import it,
// so "generated codes always pass validation" holds by construction.
package shortcode

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

// Charset for short codes: A-Z, a-z, 0-9 (62 characters).
const Charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// Length is the generated code length. Validation accepts MinLength-MaxLength
// so already-issued codes keep working if Length changes later.
// Keep MaxLength <= models.Link.ShortCode varchar(10).
const Length = 6
const MinLength = 4
const MaxLength = 10

// ErrInvalid is returned when a code fails validation.
var ErrInvalid = errors.New("invalid short code")

// lookup for O(1) charset membership (also rejects non-ASCII).
var charsetSet = func() map[byte]struct{} {
	m := make(map[byte]struct{}, len(Charset))
	for i := 0; i < len(Charset); i++ {
		m[Charset[i]] = struct{}{}
	}
	return m
}()

// Generate returns a collision-resistant random code of Length chars.
// Uses crypto/rand with rejection sampling (no modulo bias).
func Generate() (string, error) {
	charsetLen := big.NewInt(int64(len(Charset)))
	code := make([]byte, Length)
	for i := 0; i < Length; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		code[i] = Charset[n.Int64()]
	}
	return string(code), nil
}

// Validate ensures the code is MinLength-MaxLength chars from Charset.
// Keeps junk out of DB queries and makes route-swallowing safer.
func Validate(code string) error {
	if code == "" {
		return fmt.Errorf("%w: short code cannot be empty", ErrInvalid)
	}
	if len(code) < MinLength || len(code) > MaxLength {
		return fmt.Errorf("%w: must be %d-%d characters", ErrInvalid, MinLength, MaxLength)
	}
	for i := 0; i < len(code); i++ {
		if _, ok := charsetSet[code[i]]; !ok {
			return fmt.Errorf("%w: must contain only [%s] characters", ErrInvalid, Charset)
		}
	}
	return nil
}
