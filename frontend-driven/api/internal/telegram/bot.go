// Package telegram implements Telegram login via a standard bot that delivers
// a one-time code in-chat. See docs/adr/0001-telegram-bot-otp-login.md.
package telegram

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	loginTokenTTL = 10 * time.Minute
	otpTTL        = 5 * time.Minute
	maxOTPAttempts = 5
	pollTimeout   = 30 // seconds, Telegram long-poll
)

// User is the Telegram account resolved after a successful login.
type User struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
}

// DisplayName returns a human-friendly name for the Telegram user.
func (u User) DisplayName() string {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name != "" {
		return name
	}
	if u.Username != "" {
		return u.Username
	}
	return fmt.Sprintf("telegram_%d", u.ID)
}

// loginEntry is a pending login keyed by its login token.
type loginEntry struct {
	expiresAt time.Time // login-token validity

	codeSet   bool
	codeHash  string
	codeExp   time.Time
	attempts  int
	user      User
}

// Bot drives the bot-OTP login flow. All exported methods are safe for
// concurrent use.
type Bot struct {
	apiToken string
	client   *http.Client

	mu       sync.Mutex
	username string
	enabled  bool
	logins   map[string]*loginEntry
}

// NewBot constructs a Bot. Call Start to fetch the bot identity and begin
// polling for /start messages.
func NewBot(apiToken string) *Bot {
	return &Bot{
		apiToken: apiToken,
		client:   &http.Client{Timeout: (pollTimeout + 10) * time.Second},
		logins:   make(map[string]*loginEntry),
	}
}

// Start resolves the bot username via getMe and, on success, launches the
// getUpdates long-polling loop until ctx is cancelled. If the token is
// missing or invalid the bot stays disabled and the service keeps running.
func (b *Bot) Start(ctx context.Context) {
	if b.apiToken == "" {
		log.Printf("telegram: no bot token configured; telegram login disabled")
		return
	}

	me, err := b.getMe(ctx)
	if err != nil {
		log.Printf("telegram: getMe failed; telegram login disabled: %v", err)
		return
	}

	b.mu.Lock()
	b.username = me.Username
	b.enabled = true
	b.mu.Unlock()

	log.Printf("telegram: bot @%s ready; polling for /start", me.Username)
	go b.pollLoop(ctx)
}

// Enabled reports whether Telegram login is available.
func (b *Bot) Enabled() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.enabled
}

// Username returns the bot's @username (without the @).
func (b *Bot) Username() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.username
}

// CreateLogin mints a single-use login token and records a pending login.
// It returns the token and the t.me deep link the user should open.
func (b *Bot) CreateLogin() (loginToken, deepLink string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.enabled {
		return "", "", fmt.Errorf("telegram login unavailable")
	}

	b.purgeExpiredLocked()

	loginToken, err = randomToken()
	if err != nil {
		return "", "", err
	}

	b.logins[loginToken] = &loginEntry{expiresAt: time.Now().Add(loginTokenTTL)}
	deepLink = fmt.Sprintf("https://t.me/%s?start=%s", b.username, loginToken)
	return loginToken, deepLink, nil
}

// Verify checks the OTP for a login token. On success it returns the resolved
// Telegram user and consumes the login token.
func (b *Bot) Verify(loginToken, code string) (*User, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry, ok := b.logins[loginToken]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(b.logins, loginToken)
		return nil, fmt.Errorf("login expired, please start again")
	}
	if !entry.codeSet {
		return nil, fmt.Errorf("no code sent yet; open the bot and press Start")
	}
	if time.Now().After(entry.codeExp) {
		delete(b.logins, loginToken)
		return nil, fmt.Errorf("code expired, please start again")
	}
	if entry.attempts >= maxOTPAttempts {
		delete(b.logins, loginToken)
		return nil, fmt.Errorf("too many attempts, please start again")
	}

	entry.attempts++
	if subtle.ConstantTimeCompare([]byte(hashCode(code)), []byte(entry.codeHash)) != 1 {
		return nil, fmt.Errorf("invalid code")
	}

	user := entry.user
	delete(b.logins, loginToken)
	return &user, nil
}

// handleStart is called by the poller when a /start <token> message arrives.
// It generates an OTP for the matching pending login and sends it in-chat.
func (b *Bot) handleStart(ctx context.Context, loginToken string, chatID int64, from User) {
	b.mu.Lock()
	entry, ok := b.logins[loginToken]
	if !ok || time.Now().After(entry.expiresAt) {
		b.mu.Unlock()
		return
	}

	code := generateOTP()
	entry.codeSet = true
	entry.codeHash = hashCode(code)
	entry.codeExp = time.Now().Add(otpTTL)
	entry.attempts = 0
	entry.user = from
	b.mu.Unlock()

	msg := fmt.Sprintf("Your login code is: %s\n\nIt expires in 5 minutes. If you didn't request this, ignore this message.", code)
	if err := b.sendMessage(ctx, chatID, msg); err != nil {
		log.Printf("telegram: sendMessage failed: %v", err)
	}
}

func (b *Bot) purgeExpiredLocked() {
	now := time.Now()
	for k, e := range b.logins {
		if now.After(e.expiresAt) {
			delete(b.logins, k)
		}
	}
}

// --- Telegram Bot API calls ---

type tgUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (b *Bot) getMe(ctx context.Context) (tgUser, error) {
	var resp struct {
		Ok     bool   `json:"ok"`
		Result tgUser `json:"result"`
	}
	if err := b.apiGet(ctx, "getMe", nil, &resp); err != nil {
		return tgUser{}, err
	}
	if !resp.Ok {
		return tgUser{}, fmt.Errorf("getMe not ok")
	}
	return resp.Result, nil
}

func (b *Bot) sendMessage(ctx context.Context, chatID int64, text string) error {
	q := url.Values{}
	q.Set("chat_id", fmt.Sprintf("%d", chatID))
	q.Set("text", text)
	var resp struct {
		Ok bool `json:"ok"`
	}
	if err := b.apiGet(ctx, "sendMessage", q, &resp); err != nil {
		return err
	}
	if !resp.Ok {
		return fmt.Errorf("sendMessage not ok")
	}
	return nil
}

type update struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From tgUser `json:"from"`
	} `json:"message"`
}

// pollLoop long-polls getUpdates and dispatches /start commands until ctx ends.
func (b *Bot) pollLoop(ctx context.Context) {
	var offset int64
	for {
		if ctx.Err() != nil {
			return
		}

		q := url.Values{}
		q.Set("timeout", fmt.Sprintf("%d", pollTimeout))
		q.Set("offset", fmt.Sprintf("%d", offset))
		q.Set("allowed_updates", `["message"]`)

		var resp struct {
			Ok     bool     `json:"ok"`
			Result []update `json:"result"`
		}
		if err := b.apiGet(ctx, "getUpdates", q, &resp); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("telegram: getUpdates error: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		for _, u := range resp.Result {
			offset = u.UpdateID + 1
			if u.Message == nil {
				continue
			}
			text := strings.TrimSpace(u.Message.Text)
			if !strings.HasPrefix(text, "/start") {
				continue
			}
			parts := strings.Fields(text)
			if len(parts) < 2 {
				continue // /start without a login token
			}
			from := u.Message.From
			b.handleStart(ctx, parts[1], u.Message.Chat.ID, User{
				ID:        from.ID,
				Username:  from.Username,
				FirstName: from.FirstName,
				LastName:  from.LastName,
			})
		}
	}
}

func (b *Bot) apiGet(ctx context.Context, method string, q url.Values, out interface{}) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/%s", b.apiToken, method)
	if q != nil {
		endpoint += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// --- helpers ---

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}

func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}
