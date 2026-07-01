package tests

import (
	"testing"

	"github.com/user/auth-service/internal/token"
)

func TestSignAndVerifyAccessToken(t *testing.T) {
	secret := "test-secret-that-is-at-least-32-chars"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"

	signed, err := token.SignAccessToken(secret, userID, email)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	claims, err := token.VerifyAccessToken(secret, signed)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	if claims.Subject != userID {
		t.Errorf("subject: got %s, want %s", claims.Subject, userID)
	}
	if claims.Email != email {
		t.Errorf("email: got %s, want %s", claims.Email, email)
	}
}

func TestVerifyAccessTokenWrongSecret(t *testing.T) {
	signed, _ := token.SignAccessToken("secret1-that-is-at-least-32-char", "user-id", "email@test.com")
	_, err := token.VerifyAccessToken("secret2-that-is-at-least-32-char", signed)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestVerifyAccessTokenInvalid(t *testing.T) {
	_, err := token.VerifyAccessToken("secret", "not-a-jwt")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}
