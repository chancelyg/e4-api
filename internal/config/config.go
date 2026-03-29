package config

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

const defaultAdminPasswordHash = "$2a$10$4ZPgUj01QYUd/4feVvRWKebBpHeWiHJQyJABYlTcycO6LiguI.Du2"
const defaultDevAuthSecret = "your-secret-key-change-in-production"

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Site     SiteConfig     `mapstructure:"site"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
}

type AuthConfig struct {
	Username       string `mapstructure:"username"`
	Password       string `mapstructure:"password"` // bcrypt hash
	Secret         string `mapstructure:"secret"`
	TOTPSecret     string `mapstructure:"totp_secret"`
	RateLimit      int    `mapstructure:"rate_limit"`
	LockoutMinutes int    `mapstructure:"lockout_minutes"`
}

type SiteConfig struct {
	Title string `mapstructure:"title"`
}

var Cfg *Config

func Load() error {
	if err := loadDotEnvIfPresent(); err != nil {
		return err
	}

	v := viper.New()

	// Bind environment variables
	if err := bindEnv(v,
		"server.host", "E4_SERVER_HOST",
		"server.port", "E4_SERVER_PORT",
		"server.mode", "E4_SERVER_MODE",
		"database.dsn", "E4_DATABASE_DSN",
		"auth.username", "E4_AUTH_USERNAME",
		"auth.password", "E4_AUTH_PASSWORD",
		"auth.secret", "E4_AUTH_SECRET",
		"auth.totp_secret", "E4_AUTH_TOTP_SECRET",
		"auth.rate_limit", "E4_AUTH_RATE_LIMIT",
		"auth.lockout_minutes", "E4_AUTH_LOCKOUT_MINUTES",
		"site.title", "E4_SITE_TITLE",
	); err != nil {
		return err
	}

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// Set defaults
	v.SetDefault("server.host", "127.0.0.1")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "development")
	v.SetDefault("database.dsn", "./data/app.db")
	v.SetDefault("auth.secret", defaultDevAuthSecret)
	v.SetDefault("auth.username", "admin")
	// Default password: "admin" (bcrypt hash)
	v.SetDefault("auth.password", defaultAdminPasswordHash)
	v.SetDefault("auth.totp_secret", "")
	v.SetDefault("auth.rate_limit", 5)
	v.SetDefault("auth.lockout_minutes", 15)
	v.SetDefault("site.title", "E4 Diary")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	Cfg = &Config{}
	if err := v.Unmarshal(Cfg); err != nil {
		return err
	}

	Cfg.Server.Host = strings.TrimSpace(Cfg.Server.Host)
	Cfg.Server.Mode = strings.TrimSpace(Cfg.Server.Mode)
	Cfg.Auth.Username = strings.TrimSpace(Cfg.Auth.Username)
	Cfg.Auth.Secret = strings.TrimSpace(Cfg.Auth.Secret)
	Cfg.Auth.TOTPSecret = strings.TrimSpace(Cfg.Auth.TOTPSecret)
	Cfg.Site.Title = strings.TrimSpace(Cfg.Site.Title)

	if strings.EqualFold(Cfg.Server.Mode, "release") {
		if Cfg.Auth.Username == "" {
			return errors.New("release 模式必须设置非空的 auth.username")
		}
		if Cfg.Auth.Password == defaultAdminPasswordHash {
			return errors.New("release 模式禁止使用默认管理员密码哈希，请通过 .env 或环境变量覆盖 auth.password")
		}
		if Cfg.Auth.Secret == "" || Cfg.Auth.Secret == defaultDevAuthSecret {
			return errors.New("release 模式必须修改 auth.secret")
		}
	}

	return nil
}

func loadDotEnvIfPresent() error {
	if _, err := os.Stat(".env"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	return gotenv.Load(".env")
}

func bindEnv(v *viper.Viper, values ...string) error {
	for i := 0; i < len(values); i += 2 {
		if err := v.BindEnv(values[i], values[i+1]); err != nil {
			return err
		}
	}
	return nil
}
