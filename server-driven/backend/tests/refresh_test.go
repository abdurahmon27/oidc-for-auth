package tests

import (
	"testing"

	"github.com/user/auth-service/internal/token"
)

func TestGenerateRefreshToken(t *testing.T) {
	raw, hash, err := token.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(raw) != 64 { // 32 bytes hex encoded
		t.Errorf("raw length: got %d, want 64", len(raw))
	}

	if len(hash) != 64 { // SHA-256 hex encoded
		t.Errorf("hash length: got %d, want 64", len(hash))
	}

	// Hash should match
	if token.HashToken(raw) != hash {
		t.Error("hash mismatch")
	}
}

func TestRefreshTokenUniqueness(t *testing.T) {
	raw1, _, _ := token.GenerateRefreshToken()
	raw2, _, _ := token.GenerateRefreshToken()

	if raw1 == raw2 {
		t.Error("tokens should be unique")
	}
}

func TestHashTokenConsistency(t *testing.T) {
	input := "test-token-value"
	h1 := token.HashToken(input)
	h2 := token.HashToken(input)

	if h1 != h2 {
		t.Error("hashing should be deterministic")
	}
}

func TestHashTokenDifferentInputs(t *testing.T) {
	h1 := token.HashToken("token-a")
	h2 := token.HashToken("token-b")

	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}
