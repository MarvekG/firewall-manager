package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const CookieName = "fm_session"

type SessionManager struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expiresAt"`
}

func NewSessionManager(secret []byte, ttl time.Duration) *SessionManager {
	return &SessionManager{secret: secret, ttl: ttl}
}

func (m *SessionManager) Set(w http.ResponseWriter, username string, secure bool) error {
	claims := Claims{Username: username, ExpiresAt: time.Now().Add(m.ttl).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := m.sign(encodedPayload)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    encodedPayload + "." + signature,
		Path:     "/",
		Expires:  time.Unix(claims.ExpiresAt, 0),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func (m *SessionManager) Clear(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (m *SessionManager) Get(r *http.Request) (Claims, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return Claims{}, false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return Claims{}, false
	}
	if !hmac.Equal([]byte(parts[1]), []byte(m.sign(parts[0]))) {
		return Claims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, false
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, false
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return Claims{}, false
	}
	return claims, true
}

func (m *SessionManager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
