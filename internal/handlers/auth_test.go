package handlers

import (
	"bytes"
	"e4-api/internal/config"
	"e4-api/internal/db"
	"e4-api/internal/middleware"
	"e4-api/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestAuthHandler() (*AuthHandler, *echo.Echo) {
	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:auth-test-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if err := database.AutoMigrate(&models.SessionRevocation{}); err != nil {
		panic(err)
	}
	db.DB = database

	authMiddleware := middleware.NewAuthMiddleware("test-secret")
	handler := NewAuthHandler(authMiddleware)
	e := echo.New()
	return handler, e
}

func resetAuthTestState() {
	loginAttemptsMu.Lock()
	loginAttempts = make(map[string]*LoginAttempt)
	loginAttemptsMu.Unlock()

	preAuthChallengesMu.Lock()
	preAuthChallenges = make(map[string]PreAuthChallenge)
	preAuthChallengesMu.Unlock()
}

func TestLoginSuccess(t *testing.T) {
	resetAuthTestState()
	config.Cfg = &config.Config{
		Server: config.ServerConfig{Mode: "debug"},
		Auth: config.AuthConfig{
			Username: "admin",
			Password: "$2a$10$4ZPgUj01QYUd/4feVvRWKebBpHeWiHJQyJABYlTcycO6LiguI.Du2",
		},
	}

	handler, e := setupTestAuthHandler()

	loginReq := LoginRequest{
		Username: "admin",
		Password: "admin",
	}
	jsonBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Login(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "admin", response["username"])
	require.NotEmpty(t, rec.Result().Cookies())
	assert.Equal(t, middleware.SessionCookieName, rec.Result().Cookies()[0].Name)
}

func TestLoginInvalidUsername(t *testing.T) {
	resetAuthTestState()
	config.Cfg = &config.Config{
		Server: config.ServerConfig{Mode: "debug"},
		Auth: config.AuthConfig{
			Username: "admin",
			Password: "$2a$10$4ZPgUj01QYUd/4feVvRWKebBpHeWiHJQyJABYlTcycO6LiguI.Du2",
		},
	}

	handler, e := setupTestAuthHandler()

	loginReq := LoginRequest{
		Username: "wronguser",
		Password: "admin",
	}
	jsonBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Login(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLoginInvalidPassword(t *testing.T) {
	resetAuthTestState()
	config.Cfg = &config.Config{
		Server: config.ServerConfig{Mode: "debug"},
		Auth: config.AuthConfig{
			Username: "admin",
			Password: "$2a$10$4ZPgUj01QYUd/4feVvRWKebBpHeWiHJQyJABYlTcycO6LiguI.Du2",
		},
	}

	handler, e := setupTestAuthHandler()

	loginReq := LoginRequest{
		Username: "admin",
		Password: "wrongpassword",
	}
	jsonBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Login(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLoginStep2RequiresChallengeToken(t *testing.T) {
	resetAuthTestState()
	secret := "JBSWY3DPEHPK3PXP"
	config.Cfg = &config.Config{
		Server: config.ServerConfig{Mode: "debug"},
		Auth: config.AuthConfig{
			Username:   "admin",
			Password:   "$2a$10$4ZPgUj01QYUd/4feVvRWKebBpHeWiHJQyJABYlTcycO6LiguI.Du2",
			TOTPSecret: secret,
		},
	}

	handler, e := setupTestAuthHandler()
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	body := bytes.NewBufferString(fmt.Sprintf(`{"code":"%s"}`, code))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login-step2", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.LoginStep2(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.JSONEq(t, `{"error":"二步验证会话无效或已过期"}`, rec.Body.String())
}

func TestLoginStep1ReturnsChallengeTokenWhen2FAEnabled(t *testing.T) {
	resetAuthTestState()
	config.Cfg = &config.Config{
		Server: config.ServerConfig{Mode: "debug"},
		Auth: config.AuthConfig{
			Username:   "admin",
			Password:   "$2a$10$4ZPgUj01QYUd/4feVvRWKebBpHeWiHJQyJABYlTcycO6LiguI.Du2",
			TOTPSecret: "JBSWY3DPEHPK3PXP",
		},
	}

	handler, e := setupTestAuthHandler()
	loginReq := LoginStep1Request{Username: "admin", Password: "admin"}
	jsonBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login-step1", bytes.NewReader(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.LoginStep1(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, true, response["needs_2fa"])
	assert.NotEmpty(t, response["challenge_token"])
}

func TestRateLimitResetsAfterLockoutExpires(t *testing.T) {
	resetAuthTestState()
	config.Cfg = &config.Config{}
	config.Cfg.Auth.RateLimit = 1
	config.Cfg.Auth.LockoutMinutes = 2

	clientIP := fmt.Sprintf("test-%d", time.Now().UnixNano())
	allowed, _ := checkRateLimit(clientIP)
	assert.True(t, allowed)

	allowed, _ = checkRateLimit(clientIP)
	assert.False(t, allowed)

	loginAttemptsMu.Lock()
	loginAttempts[clientIP].LockedUntil = time.Now().Add(-time.Minute)
	loginAttemptsMu.Unlock()

	allowed, _ = checkRateLimit(clientIP)
	assert.True(t, allowed)
}

func TestLogout(t *testing.T) {
	resetAuthTestState()
	config.Cfg = &config.Config{Server: config.ServerConfig{Mode: "debug"}}
	handler, e := setupTestAuthHandler()

	sessionID := handler.AuthMiddleware.CreateSession("admin")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("session_id", sessionID)

	err := handler.Logout(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]bool
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.True(t, response["success"])
}

func TestStatusNotLoggedIn(t *testing.T) {
	resetAuthTestState()
	handler, e := setupTestAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Status(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.False(t, response["is_logged_in"].(bool))
}

func TestStatusLoggedIn(t *testing.T) {
	resetAuthTestState()
	handler, e := setupTestAuthHandler()

	sessionID := handler.AuthMiddleware.CreateSession("admin")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: sessionID,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Status(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)
	assert.True(t, response["is_logged_in"].(bool))
	assert.Equal(t, "admin", response["username"])
}

func TestStatusRejectsTamperedSession(t *testing.T) {
	resetAuthTestState()
	handler, e := setupTestAuthHandler()

	sessionID := handler.AuthMiddleware.CreateSession("admin") + "tampered"
	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: sessionID,
	})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Status(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.False(t, response["is_logged_in"].(bool))
}

func TestLogoutRevokesSignedSession(t *testing.T) {
	resetAuthTestState()
	config.Cfg = &config.Config{Server: config.ServerConfig{Mode: "debug"}}
	handler, e := setupTestAuthHandler()

	token := handler.AuthMiddleware.CreateSession("admin")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("session_id", token)

	err := handler.Logout(c)
	require.NoError(t, err)
	assert.Nil(t, handler.AuthMiddleware.GetSession(token))

	sessionID, ok := middleware.ExtractSessionIDForTest(token)
	require.True(t, ok)

	var revoked models.SessionRevocation
	require.NoError(t, db.DB.First(&revoked, "session_id = ?", sessionID).Error)
}

func TestRevokedSessionSurvivesNewMiddlewareInstance(t *testing.T) {
	resetAuthTestState()
	handler, _ := setupTestAuthHandler()

	token := handler.AuthMiddleware.CreateSession("admin")
	handler.AuthMiddleware.DestroySession(token)

	another := middleware.NewAuthMiddleware("test-secret")
	assert.Nil(t, another.GetSession(token))
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{name: "less than minute", input: 30 * time.Second, expected: "30秒"},
		{name: "exact minutes", input: 2 * time.Minute, expected: "2分钟"},
		{name: "rounds to nearest minute", input: 90 * time.Second, expected: "2分钟"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatDuration(tt.input))
		})
	}
}
