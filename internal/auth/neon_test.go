package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWKSURL(t *testing.T) {
	if got := JWKSURL("https://ep-x.aws.neon.tech/db/auth"); got != "https://ep-x.aws.neon.tech/db/auth/.well-known/jwks.json" {
		t.Errorf("JWKSURL = %q", got)
	}
	if got := JWKSURL("https://ep-x.aws.neon.tech/db/auth/"); got != "https://ep-x.aws.neon.tech/db/auth/.well-known/jwks.json" {
		t.Errorf("JWKSURL trailing slash = %q", got)
	}
}

func TestIssuer(t *testing.T) {
	got, err := Issuer("https://ep-x.aws.neon.tech/neondb/auth")
	if err != nil || got != "https://ep-x.aws.neon.tech" {
		t.Errorf("Issuer = %q, %v", got, err)
	}
	for _, bad := range []string{"", "not-a-url://", ":///missing"} {
		if _, err := Issuer(bad); err == nil {
			t.Errorf("Issuer(%q) expected error", bad)
		}
	}
	if _, err := Issuer("https:///nohost"); err == nil {
		t.Errorf("Issuer missing host expected error")
	}
}

func TestNewVerifierRequiresBaseURL(t *testing.T) {
	if _, err := NewVerifier(""); err == nil {
		t.Errorf("expected error for empty base URL")
	}
}

// stubJWKS serves a JWKS with the given ed25519 public key.
func stubJWKS(t *testing.T, kid string, pub ed25519.PublicKey) *httptest.Server {
	t.Helper()
	x := base64.RawURLEncoding.EncodeToString(pub)
	body, _ := json.Marshal(map[string]any{"keys": []map[string]string{
		{"kty": "OKP", "crv": "Ed25519", "kid": kid, "x": x},
	}})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func signEdDSA(t *testing.T, priv ed25519.PrivateKey, kid, issuer string, exp time.Time) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"iss": issuer,
		"sub": "user-123",
		"exp": exp.Unix(),
		"iat": time.Now().Unix(),
	})
	token.Header["kid"] = kid
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func TestValidateRoundtrip(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	jwks := stubJWKS(t, "test-key", pub)
	base := "https://ep-test.aws.neon.tech/db/auth"
	issuer := "https://ep-test.aws.neon.tech"

	v, err := NewVerifierWithJWKS(base, jwks.URL)
	if err != nil {
		t.Fatalf("NewVerifierWithJWKS: %v", err)
	}
	if v.issuer != issuer {
		t.Fatalf("issuer = %q, want %q", v.issuer, issuer)
	}
	token := signEdDSA(t, priv, "test-key", issuer, time.Now().Add(15*time.Minute))
	claims, err := v.Validate(token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if claims["sub"] != "user-123" {
		t.Errorf("sub = %v", claims["sub"])
	}
}

func TestValidateRejects(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	jwks := stubJWKS(t, "test-key", pub)
	base := "https://ep-test.aws.neon.tech/db/auth"
	issuer := "https://ep-test.aws.neon.tech"
	v, err := NewVerifierWithJWKS(base, jwks.URL)
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}

	cases := map[string]string{
		"empty":        "",
		"garbage":      "not.a.token",
		"wrong issuer": signEdDSA(t, priv, "test-key", "https://evil.example.com", time.Now().Add(15*time.Minute)),
		"expired":      signEdDSA(t, priv, "test-key", issuer, time.Now().Add(-time.Minute)),
		"unknown kid": func() string {
			token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{"iss": issuer, "sub": "u", "exp": time.Now().Add(15 * time.Minute).Unix()})
			token.Header["kid"] = "nope"
			s, _ := token.SignedString(priv)
			return s
		}(),
	}
	for name, tok := range cases {
		if _, err := v.Validate(tok); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}

	// Wrong key, correct kid/issuer.
	_, wrongPriv, _ := ed25519.GenerateKey(rand.Reader)
	bad := signEdDSA(t, wrongPriv, "test-key", issuer, time.Now().Add(15*time.Minute))
	if _, err := v.Validate(bad); err == nil {
		t.Errorf("wrong key: expected error")
	}
}
