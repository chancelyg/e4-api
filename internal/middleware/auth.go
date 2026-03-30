package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"e4-api/internal/db"
	"e4-api/internal/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const SessionCookieName = "e4_session"

const sessionTTL = 7 * 24 * time.Hour

// SessionData stores session information.
type SessionData struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	Username   string `json:"username"`
	ExpiresAt  int64  `json:"expires_at"`
}

// AuthMiddleware validates session.
type AuthMiddleware struct {
	secret []byte
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	if strings.TrimSpace(secret) == "" {
		secret = "e4-session-default-secret"
	}

	return &AuthMiddleware{secret: []byte(secret)}
}

func (a *AuthMiddleware) ValidateSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		cookie, err := c.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			log.Printf("session missing request_id=%s ip=%s path=%s", requestID, c.RealIP(), c.Path())
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "未登录或会话已过期",
			})
		}

		sessionID, session, err := a.parseSession(cookie.Value)
		if err != nil || !session.IsLoggedIn {
			log.Printf("session invalid request_id=%s ip=%s path=%s error=%v", requestID, c.RealIP(), c.Path(), err)
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "未登录或会话已过期",
			})
		}

		log.Printf("session validated request_id=%s ip=%s username=%q path=%s", requestID, c.RealIP(), session.Username, c.Path())
		c.Set("username", session.Username)
		c.Set("session_id", sessionID)
		return next(c)
	}
}

func (a *AuthMiddleware) CreateSession(username string) string {
	sessionID := generateSessionID()
	expiresAt := time.Now().Add(sessionTTL).Unix()
	payload := encodeSessionPayload(sessionID, username, expiresAt)
	signature := a.signPayload(payload)
	return payload + "." + signature
}

func (a *AuthMiddleware) DestroySession(sessionID string) {
	if sessionID == "" || db.DB == nil {
		log.Printf("session destroy skipped session_present=%t db_ready=%t", sessionID != "", db.DB != nil)
		return
	}
	if parsedSessionID, ok := extractSessionID(sessionID); ok {
		sessionID = parsedSessionID
	}

	expiresAt := time.Now().Add(sessionTTL)
	record := models.SessionRevocation{
		SessionID: sessionID,
		ExpiresAt: expiresAt,
	}

	_ = db.DB.Where("expires_at <= ?", time.Now()).Delete(&models.SessionRevocation{}).Error
	_ = db.DB.Save(&record).Error
	log.Printf("session revoked expires_at=%s", expiresAt.Format(time.RFC3339))
}

func (a *AuthMiddleware) GetSession(token string) *SessionData {
	_, session, err := a.parseSession(token)
	if err != nil {
		return nil
	}
	return session
}

func (a *AuthMiddleware) SessionMaxAgeSeconds() int {
	return int(sessionTTL.Seconds())
}

func (a *AuthMiddleware) parseSession(token string) (string, *SessionData, error) {
	separator := strings.LastIndex(token, ".")
	if separator <= 0 || separator >= len(token)-1 {
		return "", nil, errors.New("invalid session format")
	}
	payload := token[:separator]
	signature := token[separator+1:]

	if !hmac.Equal([]byte(signature), []byte(a.signPayload(payload))) {
		return "", nil, errors.New("invalid session signature")
	}

	sessionID, username, expiresAt, err := decodeSessionPayload(payload)
	if err != nil {
		return "", nil, err
	}
	if time.Now().Unix() > expiresAt {
		return "", nil, errors.New("session expired")
	}
	if a.isRevoked(sessionID) {
		return "", nil, errors.New("session revoked")
	}

	return sessionID, &SessionData{
		IsLoggedIn: true,
		Username:   username,
		ExpiresAt:  expiresAt,
	}, nil
}

func (a *AuthMiddleware) signPayload(payload string) string {
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *AuthMiddleware) isRevoked(sessionID string) bool {
	if db.DB == nil {
		return false
	}

	now := time.Now()
	_ = db.DB.Where("expires_at <= ?", now).Delete(&models.SessionRevocation{}).Error

	var record models.SessionRevocation
	err := db.DB.First(&record, "session_id = ?", sessionID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false
	}
	if err != nil {
		log.Printf("session revocation lookup failed: %v", err)
	}
	return err == nil
}

func encodeSessionPayload(sessionID, username string, expiresAt int64) string {
	encodedSessionID := base64.RawURLEncoding.EncodeToString([]byte(sessionID))
	encodedUser := base64.RawURLEncoding.EncodeToString([]byte(username))
	return strings.Join([]string{encodedSessionID, encodedUser, strconv.FormatInt(expiresAt, 10)}, ".")
}

func decodeSessionPayload(payload string) (string, string, int64, error) {
	parts := strings.Split(payload, ".")
	if len(parts) != 3 {
		return "", "", 0, errors.New("invalid session payload")
	}

	sessionIDBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", 0, err
	}
	usernameBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", 0, err
	}
	expiresAt, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return "", "", 0, err
	}

	return string(sessionIDBytes), string(usernameBytes), expiresAt, nil
}

func extractSessionID(value string) (string, bool) {
	separator := strings.LastIndex(value, ".")
	if separator <= 0 || separator >= len(value)-1 {
		return "", false
	}

	sessionID, _, _, err := decodeSessionPayload(value[:separator])
	if err != nil {
		return "", false
	}
	return sessionID, true
}

func ExtractSessionIDForTest(value string) (string, bool) {
	return extractSessionID(value)
}

func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("generate session id: %v", err))
	}
	return hex.EncodeToString(b)
}
