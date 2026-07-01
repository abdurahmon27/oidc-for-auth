package telegram

import (
	"testing"
	"time"
)

// newTestBot returns an enabled bot with no network dependency.
func newTestBot() *Bot {
	b := NewBot("test-token")
	b.enabled = true
	b.username = "testbot"
	return b
}

// seedCode simulates what the poller's handleStart does after /start, without
// hitting the Telegram API: it attaches an OTP to a pending login.
func (b *Bot) seedCode(loginToken, code string, user User) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.logins[loginToken]
	e.codeSet = true
	e.codeHash = hashCode(code)
	e.codeExp = time.Now().Add(otpTTL)
	e.attempts = 0
	e.user = user
}

func TestVerifyHappyPath(t *testing.T) {
	b := newTestBot()
	loginToken, deepLink, err := b.CreateLogin()
	if err != nil {
		t.Fatalf("CreateLogin: %v", err)
	}
	if deepLink == "" || loginToken == "" {
		t.Fatal("expected non-empty login token and deep link")
	}

	want := User{ID: 42, Username: "alice", FirstName: "Alice"}
	b.seedCode(loginToken, "123456", want)

	got, err := b.Verify(loginToken, "123456")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("got user id %d, want %d", got.ID, want.ID)
	}

	// Single-use: token is consumed.
	if _, err := b.Verify(loginToken, "123456"); err == nil {
		t.Fatal("expected error reusing consumed login token")
	}
}

func TestVerifyWrongCodeAndAttemptCap(t *testing.T) {
	b := newTestBot()
	loginToken, _, _ := b.CreateLogin()
	b.seedCode(loginToken, "654321", User{ID: 1})

	for i := 0; i < maxOTPAttempts; i++ {
		if _, err := b.Verify(loginToken, "000000"); err == nil {
			t.Fatalf("attempt %d: expected invalid-code error", i)
		}
	}
	// After maxOTPAttempts wrong tries the login is invalidated, so even the
	// correct code must fail.
	if _, err := b.Verify(loginToken, "654321"); err == nil {
		t.Fatal("expected lockout after too many attempts")
	}
}

func TestVerifyExpiredCode(t *testing.T) {
	b := newTestBot()
	loginToken, _, _ := b.CreateLogin()
	b.seedCode(loginToken, "111111", User{ID: 2})

	b.mu.Lock()
	b.logins[loginToken].codeExp = time.Now().Add(-time.Second)
	b.mu.Unlock()

	if _, err := b.Verify(loginToken, "111111"); err == nil {
		t.Fatal("expected expired-code error")
	}
}

func TestVerifyBeforeCodeSent(t *testing.T) {
	b := newTestBot()
	loginToken, _, _ := b.CreateLogin()
	// No seedCode: user hasn't pressed Start yet.
	if _, err := b.Verify(loginToken, "123456"); err == nil {
		t.Fatal("expected error when no code has been sent")
	}
}

func TestVerifyUnknownToken(t *testing.T) {
	b := newTestBot()
	if _, err := b.Verify("does-not-exist", "123456"); err == nil {
		t.Fatal("expected error for unknown login token")
	}
}

func TestCreateLoginDisabled(t *testing.T) {
	b := NewBot("") // never Started, stays disabled
	if _, _, err := b.CreateLogin(); err == nil {
		t.Fatal("expected error creating login while disabled")
	}
}
