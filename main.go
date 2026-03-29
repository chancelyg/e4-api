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
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "E4 Diary\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "一个单机部署的个人日记服务，内置 Web 界面与 JSON API。\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "用法:\n  %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n版本信息:\n  version: %s\n  commit: %s\n  built: %s\n", version, commit, buildDate)
		fmt.Fprintf(flag.CommandLine.Output(), "\n常用环境变量:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_SERVER_PORT            监听端口\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_SERVER_MODE            运行模式（development/release）\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_DATABASE_DSN           SQLite 数据库路径\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_AUTH_USERNAME          登录用户名\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_AUTH_PASSWORD          bcrypt 密码哈希\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  E4_AUTH_SECRET            会话签名密钥\n")
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

	// Initialize database
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))

	// Initialize auth middleware and handlers
	authMiddleware := middleware.NewAuthMiddleware(config.Cfg.Auth.Secret)
	authHandler := handlers.NewAuthHandler(authMiddleware)
	diaryHandler := handlers.NewDiaryHandler()
	goalHandler := handlers.NewGoalHandler()
	commonHandler := handlers.NewCommonHandler()

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

		// Protected routes
		protected := api.Group("", authMiddleware.ValidateSession)
		protected.GET("/diary", diaryHandler.List)
		protected.POST("/diary", diaryHandler.Create)
		protected.GET("/diary/stats", diaryHandler.Stats)
		protected.GET("/diary/:id", diaryHandler.Get)
		protected.PUT("/diary/:id", diaryHandler.Update)
		protected.DELETE("/diary/:id", diaryHandler.Delete)
		protected.GET("/goals", goalHandler.List)
		protected.POST("/goals", goalHandler.Create)
		protected.GET("/goals/dashboard", goalHandler.Dashboard)
		protected.PUT("/goals/:id", goalHandler.Update)
		protected.DELETE("/goals/:id", goalHandler.Delete)
		protected.PUT("/goals/:id/records/:date", goalHandler.UpsertRecord)
		protected.DELETE("/goals/:id/records/:date", goalHandler.DeleteRecord)
		protected.GET("/ip", commonHandler.GetIP)
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

	log.Printf("Server starting on port %d", port)
	log.Fatal(e.Start(":" + strconv.Itoa(port)))
}
