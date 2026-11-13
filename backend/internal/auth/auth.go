// Package auth issues and validates credentials.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Role mirrors the user_role enum.
type Role string

const (
	RoleAdmin     Role = "ADMIN"
	RoleEvaluator Role = "EVALUATOR"
	RoleStudent   Role = "STUDENT"
)

var ErrInvalidToken = errors.New("invalid or expired token")

const (
	PurposeAccess  = "access"
	PurposeRefresh = "refresh"
	PurposePDF     = "pdf"
)

// Claims is the access-token payload.
type Claims struct {
	UserID   string `json:"uid"`
	Role     Role   `json:"role"`
	Purpose  string `json:"pur"`
	ScriptID string `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

// Manager signs tokens with one secret and fixed policy TTLs.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewManager(secret []byte, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: secret, accessTTL: accessTTL, refreshTTL: refreshTTL, issuer: "exam-evaluation"}
}

// HashPassword derives a bcrypt digest.
func HashPassword(pw string) (string, error) {
	if len(pw) < 10 {
		return "", errors.New("password must be at least 10 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

// CheckPassword verifies a digest.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func (m *Manager) sign(c Claims, ttl time.Duration) (string, error) {
	now := time.Now()
	c.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    m.issuer,
		Subject:   c.UserID,
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

// Access mints a short-lived access token.
func (m *Manager) Access(userID string, role Role) (string, error) {
	return m.sign(Claims{UserID: userID, Role: role, Purpose: PurposeAccess}, m.accessTTL)
}

// Refresh mints a long-lived refresh token plus its storage hash.
func (m *Manager) Refresh(userID string, role Role) (token, hash string, expires time.Time, err error) {
	token, err = m.sign(Claims{UserID: userID, Role: role, Purpose: PurposeRefresh}, m.refreshTTL)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return token, HashToken(token), time.Now().Add(m.refreshTTL), nil
}

// PDFToken mints a token that authorises one script's PDF for a short window.
// Needed because <iframe>/<embed> cannot send an Authorization header.
func (m *Manager) PDFToken(userID, scriptID string) (string, error) {
	return m.sign(Claims{UserID: userID, Purpose: PurposePDF, ScriptID: scriptID}, 10*time.Minute)
}

// Parse validates signature, issuer, purpose and expiry.
func (m *Manager) Parse(raw, purpose string) (*Claims, error) {
	var c Claims
	_, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, ErrInvalidToken
	}
	if c.Purpose != purpose {
		return nil, ErrInvalidToken
	}
	return &c, nil
}

// HashToken stores refresh tokens hashed, never in the clear.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// RandomSecret produces a URL-safe random string (used for opaque ids).
func RandomSecret(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
