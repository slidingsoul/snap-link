// Package auth verifies Neon Managed Better Auth JWTs.
//
// Server needs only NEON_AUTH_BASE_URL (per-branch, from Neon Console
// Branch → Auth page). JWKS URL and expected issuer are derived:
// JWKS = <base>/.well-known/jwks.json, issuer = origin of <base>.
// Verification is issuer-only (no audience check) per Neon's Go recipe:
// EdDSA signature + issuer + expiry + kid match.
package auth

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Verifier validates Neon JWTs against the branch JWKS.
type Verifier struct {
	jwks   keyfunc.Keyfunc
	issuer string
}

// JWKSURL derives the JWKS endpoint from the auth base URL.
func JWKSURL(baseURL string) string {
	return strings.TrimSuffix(baseURL, "/") + "/.well-known/jwks.json"
}

// Issuer derives the expected issuer (origin) from the auth base URL.
// e.g. https://ep-xx.aws.neon.tech/neondb/auth -> https://ep-xx.aws.neon.tech
func Issuer(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid auth base URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid auth base URL %q: missing scheme or host", baseURL)
	}
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}

// NewVerifier builds a verifier with a cached remote JWKS.
func NewVerifier(baseURL string) (*Verifier, error) {
	return NewVerifierWithJWKS(baseURL, "")
}

// NewVerifierWithJWKS is NewVerifier with an explicit JWKS URL override.
// Pass "" to derive from baseURL. Used by tests to point at a stub server.
func NewVerifierWithJWKS(baseURL, jwksURL string) (*Verifier, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("NEON_AUTH_BASE_URL is required")
	}
	issuer, err := Issuer(baseURL)
	if err != nil {
		return nil, err
	}
	if jwksURL == "" {
		jwksURL = JWKSURL(baseURL)
	}
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to load JWKS from %s: %w", jwksURL, err)
	}
	return &Verifier{jwks: jwks, issuer: issuer}, nil
}

// Validate parses and verifies tokenString, returning claims on success.
// Checks: EdDSA signature via JWKS, issuer match, expiry (no audience).
func (v *Verifier) Validate(tokenString string) (jwt.MapClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, fmt.Errorf("missing bearer token")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return v.jwks.Keyfunc(token)
	},
		jwt.WithIssuer(v.issuer),
		jwt.WithValidMethods([]string{"EdDSA"}),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	if claims["sub"] == "" {
		return nil, fmt.Errorf("invalid token: missing sub claim")
	}
	return claims, nil
}
