package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"e4-api/internal/config"
	"e4-api/internal/middleware"

	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	AuthMiddleware *middleware.AuthMiddleware
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginStep1Request struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginStep2Request struct {
	Code           string `json:"code" validate:"required"`
	ChallengeToken string `json:"challenge_token"`
}

type LoginAttempt struct {
	Count        int
	FirstAttempt time.Time
	LockedUntil  time.Time
}

type PreAuthChallenge struct {
	Username  string
	ExpiresAt time.Time
}

const (
	rateLimitWindow = 15 * time.Minute
	challengeTTL    = 5 * time.Minute
)

var (
	loginAttempts   = make(map[string]*LoginAttempt)
	loginAttemptsMu sync.Mutex

	preAuthChallenges   = make(map[string]PreAuthChallenge)
	preAuthChallengesMu sync.Mutex
)

func NewAuthHandler(am *middleware.AuthMiddleware) *AuthHandler {
	return &AuthHandler{
		AuthMiddleware: am,
	}
}

func getClientIP(c echo.Context) string {
	return c.RealIP()
}

func checkRateLimit(clientIP string) (bool, string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()

	attempt, exists := loginAttempts[clientIP]
	now := time.Now()

	if !exists {
		loginAttempts[clientIP] = &LoginAttempt{
			Count:        1,
			FirstAttempt: now,
		}
		return true, ""
	}

	if !attempt.LockedUntil.IsZero() && !attempt.LockedUntil.After(now) {
		attempt.Count = 0
		attempt.FirstAttempt = now
		attempt.LockedUntil = time.Time{}
	}

	if now.Sub(attempt.FirstAttempt) > rateLimitWindow {
		attempt.Count = 0
		attempt.FirstAttempt = now
	}

	if attempt.LockedUntil.After(now) {
		remaining := attempt.LockedUntil.Sub(now)
		return false, "登录尝试过于频繁，请 " + formatDuration(remaining) + " 后重试"
	}

	rateLimit := config.Cfg.Auth.RateLimit
	if rateLimit <= 0 {
		rateLimit = 5
	}

	if attempt.Count >= rateLimit {
		lockoutMinutes := config.Cfg.Auth.LockoutMinutes
		if lockoutMinutes <= 0 {
			lockoutMinutes = 15
		}
		attempt.LockedUntil = now.Add(time.Duration(lockoutMinutes) * time.Minute)
		return false, "登录尝试次数过多，请 " + formatDuration(attempt.LockedUntil.Sub(now)) + " 后重试"
	}

	if attempt.Count == 0 {
		attempt.FirstAttempt = now
	}
	attempt.Count++
	return true, ""
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "30秒"
	}
	minutes := int(d.Round(time.Minute).Minutes())
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf("%d分钟", minutes)
}

func resetLoginAttempts(clientIP string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	delete(loginAttempts, clientIP)
}

func createChallengeToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func savePreAuthChallenge(username string) (string, error) {
	token, err := createChallengeToken()
	if err != nil {
		return "", err
	}

	preAuthChallengesMu.Lock()
	defer preAuthChallengesMu.Unlock()

	now := time.Now()
	for key, challenge := range preAuthChallenges {
		if challenge.ExpiresAt.Before(now) {
			delete(preAuthChallenges, key)
		}
	}

	preAuthChallenges[token] = PreAuthChallenge{
		Username:  username,
		ExpiresAt: now.Add(challengeTTL),
	}

	return token, nil
}

func consumePreAuthChallenge(token string) (string, bool) {
	if token == "" {
		return "", false
	}

	preAuthChallengesMu.Lock()
	defer preAuthChallengesMu.Unlock()

	challenge, exists := preAuthChallenges[token]
	if !exists {
		return "", false
	}
	delete(preAuthChallenges, token)

	if challenge.ExpiresAt.Before(time.Now()) {
		return "", false
	}

	return challenge.Username, true
}

func (h *AuthHandler) LoginStep1(c echo.Context) error {
	clientIP := getClientIP(c)

	allowed, msg := checkRateLimit(clientIP)
	if !allowed {
		return c.JSON(http.StatusTooManyRequests, map[string]string{
			"error": msg,
		})
	}

	req := new(LoginStep1Request)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "无效的请求数据",
		})
	}

	if req.Username != config.Cfg.Auth.Username {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "用户名或密码错误",
		})
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(config.Cfg.Auth.Password),
		[]byte(req.Password),
	); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "用户名或密码错误",
		})
	}

	if config.Cfg.Auth.TOTPSecret != "" {
		challengeToken, err := savePreAuthChallenge(req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "创建验证挑战失败",
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"needs_2fa":       true,
			"challenge_token": challengeToken,
		})
	}

	sessionID := h.AuthMiddleware.CreateSession(req.Username)
	setSessionCookie(c, sessionID, h.AuthMiddleware.SessionMaxAgeSeconds())
	resetLoginAttempts(clientIP)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"username": req.Username,
	})
}

func (h *AuthHandler) LoginStep2(c echo.Context) error {
	clientIP := getClientIP(c)

	allowed, msg := checkRateLimit(clientIP)
	if !allowed {
		return c.JSON(http.StatusTooManyRequests, map[string]string{
			"error": msg,
		})
	}

	code := c.FormValue("code")
	challengeToken := c.FormValue("challenge_token")
	if code == "" {
		req := new(LoginStep2Request)
		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "无效的请求数据",
			})
		}
		code = req.Code
		challengeToken = req.ChallengeToken
	}

	if config.Cfg.Auth.TOTPSecret == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "未配置二步验证",
		})
	}

	username, ok := consumePreAuthChallenge(challengeToken)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "二步验证会话无效或已过期",
		})
	}

	if valid := totp.Validate(code, config.Cfg.Auth.TOTPSecret); !valid {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "验证码错误",
		})
	}

	sessionID := h.AuthMiddleware.CreateSession(username)
	setSessionCookie(c, sessionID, h.AuthMiddleware.SessionMaxAgeSeconds())
	resetLoginAttempts(clientIP)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"username": username,
	})
}

func setSessionCookie(c echo.Context, sessionID string, maxAge int) {
	cookie := new(http.Cookie)
	cookie.Name = middleware.SessionCookieName
	cookie.Value = sessionID
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Secure = config.Cfg.Server.Mode == "release"
	cookie.MaxAge = maxAge
	cookie.Expires = time.Now().Add(time.Duration(cookie.MaxAge) * time.Second)
	c.SetCookie(cookie)
}

func (h *AuthHandler) Login(c echo.Context) error {
	clientIP := getClientIP(c)

	allowed, msg := checkRateLimit(clientIP)
	if !allowed {
		return c.JSON(http.StatusTooManyRequests, map[string]string{
			"error": msg,
		})
	}

	req := new(LoginRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "无效的请求数据",
		})
	}

	if req.Username != config.Cfg.Auth.Username {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "用户名或密码错误",
		})
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(config.Cfg.Auth.Password),
		[]byte(req.Password),
	); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "用户名或密码错误",
		})
	}

	if config.Cfg.Auth.TOTPSecret != "" {
		challengeToken, err := savePreAuthChallenge(req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "创建验证挑战失败",
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"needs_2fa":       true,
			"challenge_token": challengeToken,
		})
	}

	sessionID := h.AuthMiddleware.CreateSession(req.Username)
	setSessionCookie(c, sessionID, h.AuthMiddleware.SessionMaxAgeSeconds())
	resetLoginAttempts(clientIP)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"username": req.Username,
	})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	sessionID, ok := c.Get("session_id").(string)
	if (!ok || sessionID == "") && c != nil {
		if cookie, err := c.Cookie(middleware.SessionCookieName); err == nil {
			sessionID = cookie.Value
		}
	}
	if sessionID != "" {
		h.AuthMiddleware.DestroySession(sessionID)
	}

	cookie := new(http.Cookie)
	cookie.Name = middleware.SessionCookieName
	cookie.Value = ""
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Secure = config.Cfg.Server.Mode == "release"
	cookie.MaxAge = -1
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]bool{
		"success": true,
	})
}

func (h *AuthHandler) Status(c echo.Context) error {
	cookie, err := c.Cookie(middleware.SessionCookieName)
	if err != nil || cookie.Value == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"is_logged_in": false,
		})
	}

	session := h.AuthMiddleware.GetSession(cookie.Value)
	if session == nil || !session.IsLoggedIn {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"is_logged_in": false,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"is_logged_in": true,
		"username":     session.Username,
	})
}
