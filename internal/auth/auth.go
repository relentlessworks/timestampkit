package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Auth handles OTP-based authentication and bearer token validation.
type Auth struct {
	secret string
	mu     sync.RWMutex
	otps   map[string]string // email -> code
	tokens map[string]string // token -> email
}

// New creates a new Auth instance.
func New(secret string) *Auth {
	return &Auth{
		secret: secret,
		otps:   make(map[string]string),
		tokens: make(map[string]string),
	}
}

// RequestOTP generates a 6-digit OTP for the given email.
// In production, this would send an email. For dev, it logs to stderr.
func (a *Auth) RequestOTP(email string) string {
	code := generateOTP()
	a.mu.Lock()
	a.otps[email] = code
	a.mu.Unlock()
	fmt.Fprintf(os.Stderr, "OTP for %s: %s\n", email, code)
	return code
}

// VerifyOTP checks the OTP and returns a long-lived bearer token.
func (a *Auth) VerifyOTP(email, code string) (string, error) {
	a.mu.RLock()
	stored, ok := a.otps[email]
	a.mu.RUnlock()
	if !ok || stored != code {
		return "", fmt.Errorf("invalid OTP code")
	}
	a.mu.Lock()
	delete(a.otps, email)
	token := generateToken(a.secret, email)
	a.tokens[token] = email
	a.mu.Unlock()
	return token, nil
}

// ValidateToken checks if a bearer token is valid and returns the email.
func (a *Auth) ValidateToken(token string) (string, bool) {
	a.mu.RLock()
	email, ok := a.tokens[token]
	a.mu.RUnlock()
	return email, ok
}

// ExtractBearer pulls the token from an Authorization header.
func ExtractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func generateOTP() string {
	b := make([]byte, 3)
	rand.Read(b)
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", n%1000000)
}

func generateToken(secret, email string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(email))
	mac.Write([]byte(time.Now().Format(time.RFC3339Nano)))
	b := make([]byte, 8)
	rand.Read(b)
	mac.Write(b)
	return hex.EncodeToString(mac.Sum(nil))
}
