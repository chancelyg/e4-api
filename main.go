package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"e4-api/internal/config"
	"e4-api/internal/db"
	"e4-api/internal/handlers"
	"e4-api/internal/middleware"
	"e4-api/pkg"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC | log.Lshortfile)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "E4 Diary\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "一个单机部署的个人日记服务，内置 Web 界面与 JSON API。\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "用法:\n  %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n版本信息:\n  version: %s\n  commit: %s\n  built: %s\n", version, commit, buildDate)
		fmt.Fprintf(flag.CommandLine.Output(), "\n常用环境变量:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_SERVER_PORT            监听端口\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_SERVER_HOST            监听地址（默认 127.0.0.1）\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_SERVER_MODE            运行模式（development/release）\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_DATABASE_DSN           SQLite 数据库路径\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_AUTH_SECRET            会话签名密钥\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\n配置建议:\n  config.yaml: 稳定业务配置（数据库、管理员用户名、bcrypt 密码哈希、2FA 配置等）\n  .env / 环境变量: 部署监听配置与敏感密钥（server.*、auth.secret）\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\n配置优先级:\n  显式环境变量 > .env > config.yaml > 内置默认值\n")
	}

	flag.Bool("version", false, "显示版本信息")
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatal(err)
	}

	if versionFlag := flag.Lookup("version"); versionFlag != nil && versionFlag.Value.String() == "true" {
		fmt.Printf("version=%s commit=%s built=%s\n", version, commit, buildDate)
		return
	}

	// Load configuration
	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	logConfigSummary()

	// Initialize database
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Printf("database initialized dsn=%q", config.Cfg.Database.DSN)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Debug = config.Cfg.Server.Mode != "release"
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		log.Printf("http error request_id=%s method=%s path=%s ip=%s error=%v", requestID, c.Request().Method, c.Path(), c.RealIP(), err)
		e.DefaultHTTPErrorHandler(err, c)
	}

	// Middleware
	e.Use(echoMiddleware.RequestID())
	e.Use(echoMiddleware.RequestLoggerWithConfig(echoMiddleware.RequestLoggerConfig{
		LogLatency:       true,
		LogProtocol:      true,
		LogRemoteIP:      true,
		LogHost:          true,
		LogMethod:        true,
		LogURI:           true,
		LogRoutePath:     true,
		LogRequestID:     true,
		LogUserAgent:     true,
		LogStatus:        true,
		LogError:         true,
		LogContentLength: true,
		LogResponseSize:  true,
		LogValuesFunc: func(c echo.Context, v echoMiddleware.RequestLoggerValues) error {
			uri := v.URI
			if shouldRedactRequestURI(c.Path()) {
				uri = redactPath(c.Path())
			}
			log.Printf(
				"http request request_id=%s ip=%s host=%s method=%s uri=%s route=%s status=%d latency=%s request_bytes=%s response_bytes=%d protocol=%s user_agent=%q error=%v",
				v.RequestID,
				v.RemoteIP,
				v.Host,
				v.Method,
				uri,
				v.RoutePath,
				v.Status,
				v.Latency.Round(time.Microsecond),
				v.ContentLength,
				v.ResponseSize,
				v.Protocol,
				v.UserAgent,
				v.Error,
			)
			return nil
		},
	}))
	e.Use(echoMiddleware.RecoverWithConfig(echoMiddleware.RecoverConfig{
		StackSize: 4 << 10,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)
			log.Printf("panic recovered request_id=%s method=%s path=%s ip=%s error=%v stack=%s", requestID, c.Request().Method, c.Path(), c.RealIP(), err, string(stack))
			return err
		},
	}))
	e.Use(echoMiddleware.Secure())
	if config.Cfg.Server.Mode != "release" {
		e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
			AllowOrigins:     []string{"http://localhost:5173"},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
			AllowCredentials: true,
		}))
	}

	// Initialize auth middleware and handlers
	authMiddleware := middleware.NewAuthMiddleware(config.Cfg.Auth.Secret)
	authHandler := handlers.NewAuthHandler(authMiddleware)
	diaryHandler := handlers.NewDiaryHandler()
	goalHandler := handlers.NewGoalHandler()
	commonHandler := handlers.NewCommonHandler()
	jsonStoreHandler := handlers.NewJSONStoreHandler()

	// Static files - use embedded filesystem (no auth required)
	// Strip the "dist" prefix from embedded paths
	webFS, err := fs.Sub(pkg.WebFS, "dist")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}

	// Serve all static assets (covers /_app/, /assets/, etc.)
	staticHandler := http.FileServer(http.FS(webFS))
	e.GET("/_app/*", echo.WrapHandler(staticHandler))
	e.GET("/assets/*", echo.WrapHandler(staticHandler))
	e.GET("/favicon.png", echo.WrapHandler(staticHandler))
	e.GET("/favicon.svg", echo.WrapHandler(staticHandler))
	e.GET("/favicon.ico", echo.WrapHandler(staticHandler))
	e.GET("/robots.txt", echo.WrapHandler(staticHandler))

	// API routes (auth middleware applied to protected routes)
	api := e.Group("/api")
	{
		// Public auth endpoints
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/login-step1", authHandler.LoginStep1)
		api.POST("/auth/login-step2", authHandler.LoginStep2)
		api.POST("/auth/logout", authHandler.Logout)
		api.GET("/auth/status", authHandler.Status)
		api.POST("/json/:key", jsonStoreHandler.Create)
		api.GET("/json/:key", jsonStoreHandler.Get)
		api.PUT("/json/:key", jsonStoreHandler.Upsert)
		api.DELETE("/json/:key", jsonStoreHandler.Delete)

		// Protected routes
		protected := api.Group("", authMiddleware.ValidateSession)
		protected.GET("/diary", diaryHandler.List)
		protected.POST("/diary", diaryHandler.Create)
		protected.GET("/diary/stats", diaryHandler.Stats)
		protected.GET("/diary/:id", diaryHandler.Get)
		protected.GET("/goals", goalHandler.List)
		protected.POST("/goals", goalHandler.Create)
		protected.GET("/goals/dashboard", goalHandler.Dashboard)
		protected.GET("/goals/year-summary", goalHandler.YearSummary)
		protected.PUT("/goals/:id", goalHandler.Update)
		protected.DELETE("/goals/:id", goalHandler.Delete)
		protected.PUT("/goals/:id/records/:date", goalHandler.UpsertRecord)
		protected.DELETE("/goals/:id/records/:date", goalHandler.DeleteRecord)
		protected.GET("/ip", commonHandler.GetIP)
		protected.GET("/admin/json", jsonStoreHandler.AdminList)
		protected.GET("/admin/json/:key/content", jsonStoreHandler.AdminGetContent)
		protected.DELETE("/admin/json/:key", jsonStoreHandler.AdminDelete)
	}

	// Serve index.html for all other routes (SPA fallback)
	e.GET("/*", func(c echo.Context) error {
		content, err := webFS.Open("index.html")
		if err != nil {
			return c.String(http.StatusNotFound, "index.html not found")
		}
		defer content.Close()
		return c.Stream(http.StatusOK, "text/html", content)
	})

	// Start server
	port := config.Cfg.Server.Port
	if port == 0 {
		port = 8080
	}

	host := config.Cfg.Server.Host
	if host == "" {
		host = "127.0.0.1"
	}

	address := host + ":" + strconv.Itoa(port)
	log.Printf("server starting address=%s mode=%s version=%s commit=%s built=%s", address, config.Cfg.Server.Mode, version, commit, buildDate)
	log.Fatal(e.Start(address))
}

func shouldRedactRequestURI(routePath string) bool {
	return strings.HasPrefix(routePath, "/api/json/") || strings.HasPrefix(routePath, "/api/admin/json/")
}

func redactPath(routePath string) string {
	if routePath == "" {
		return ""
	}
	return routePath
}

func logConfigSummary() {
	log.Printf(
		"config loaded server.host=%q(source=%s) server.port=%d(source=%s) server.mode=%q(source=%s) database.dsn=%q(source=%s) auth.username=%q(source=%s) auth.password.source=%s auth.password.default=%t auth.secret.source=%s auth.secret.default=%t auth.totp.enabled=%t auth.totp.source=%s auth.rate_limit=%d(source=%s) auth.lockout_minutes=%d(source=%s) site.title=%q(source=%s)",
		config.Cfg.Server.Host,
		config.Source("server.host"),
		config.Cfg.Server.Port,
		config.Source("server.port"),
		config.Cfg.Server.Mode,
		config.Source("server.mode"),
		config.Cfg.Database.DSN,
		config.Source("database.dsn"),
		config.Cfg.Auth.Username,
		config.Source("auth.username"),
		config.Source("auth.password"),
		config.UsesDefaultAdminPasswordHash(),
		config.Source("auth.secret"),
		config.UsesDefaultDevAuthSecret(),
		config.Cfg.Auth.TOTPSecret != "",
		config.Source("auth.totp_secret"),
		config.Cfg.Auth.RateLimit,
		config.Source("auth.rate_limit"),
		config.Cfg.Auth.LockoutMinutes,
		config.Source("auth.lockout_minutes"),
		config.Cfg.Site.Title,
		config.Source("site.title"),
	)
}
