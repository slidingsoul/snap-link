package shortcode

import (
	"errors"
	"testing"
)

func TestValidate(t *testing.T) {
	valid := []string{"abc123", "Kp8mQw", "abcd", "1234567890", "ABCDEF"}
	for _, c := range valid {
		if err := Validate(c); err != nil {
			t.Errorf("Validate(%q) unexpected error: %v", c, err)
		}
	}
	invalid := []string{"", "ab", "abc", "toolongcode123", "ab c12", "ab-c12", "ab/c12", "ab_c12", "äbc123"}
	for _, c := range invalid {
		if err := Validate(c); err == nil {
			t.Errorf("Validate(%q) expected error, got nil", c)
		} else if !errors.Is(err, ErrInvalid) {
			t.Errorf("Validate(%q) error should wrap ErrInvalid, got: %v", c, err)
		}
	}
}

func TestGenerate(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		code, err := Generate()
		if err != nil {
			t.Fatalf("Generate() error: %v", err)
		}
		if len(code) != Length {
			t.Fatalf("Generate() length = %d, want %d", len(code), Length)
		}
		if err := Validate(code); err != nil {
			t.Fatalf("Generate() produced invalid code %q: %v", code, err)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("Generate() duplicate %q in 100 draws (unlikely, check RNG)", code)
		}
		seen[code] = struct{}{}
	}
}
