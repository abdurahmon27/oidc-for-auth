package tests

import (
	"testing"

	"github.com/user/auth-service/internal/auth"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := auth.NewRegistry()
	fb := auth.NewFacebookProvider("id", "secret", "http://localhost/callback")
	registry.Register(fb)

	p, err := registry.Get("facebook")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if p.Name() != "facebook" {
		t.Errorf("name: got %s, want facebook", p.Name())
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	registry := auth.NewRegistry()
	_, err := registry.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestRegistryList(t *testing.T) {
	registry := auth.NewRegistry()
	registry.Register(auth.NewFacebookProvider("id", "secret", "http://localhost/callback"))
	registry.Register(auth.NewGitHubProvider("id", "secret", "http://localhost/callback"))

	names := registry.List()
	if len(names) != 2 {
		t.Errorf("list length: got %d, want 2", len(names))
	}
}

func TestPKCECodeVerifier(t *testing.T) {
	v, err := auth.GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(v) == 0 {
		t.Error("verifier should not be empty")
	}
}

func TestStateGenerateAndVerify(t *testing.T) {
	secret := "test-secret-for-state-generation"
	state, err := auth.GenerateState(secret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if err := auth.VerifyState(secret, state); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestStateVerifyWrongSecret(t *testing.T) {
	state, _ := auth.GenerateState("secret1")
	err := auth.VerifyState("secret2", state)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}
