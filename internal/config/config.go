package config

import (
	"errors"
	"strings"

	"github.com/spf13/viper"
)

const defaultAdminPasswordHash = "$2a$10$4ZPgUj01QYUd/4feVvRWKebBpHeWiHJQyJABYlTcycO6LiguI.Du2"

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Site     SiteConfig     `mapstructure:"site"`
}

type ServerConfig struct {
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
	// Bind environment variables
	viper.BindEnv("server.port", "E4_SERVER_PORT")
	viper.BindEnv("server.mode", "E4_SERVER_MODE")
	viper.BindEnv("database.dsn", "E4_DATABASE_DSN")
	viper.BindEnv("auth.username", "E4_AUTH_USERNAME")
	viper.BindEnv("auth.password", "E4_AUTH_PASSWORD")
	viper.BindEnv("auth.secret", "E4_AUTH_SECRET")
	viper.BindEnv("auth.totp_secret", "E4_AUTH_TOTP_SECRET")
	viper.BindEnv("auth.rate_limit", "E4_AUTH_RATE_LIMIT")
	viper.BindEnv("auth.lockout_minutes", "E4_AUTH_LOCKOUT_MINUTES")
	viper.BindEnv("site.title", "E4_SITE_TITLE")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "development")
	viper.SetDefault("database.dsn", "./data/app.db")
	viper.SetDefault("auth.secret", "your-secret-key-change-in-production")
	viper.SetDefault("auth.username", "admin")
	// Default password: "admin" (bcrypt hash)
	viper.SetDefault("auth.password", defaultAdminPasswordHash)
	viper.SetDefault("auth.totp_secret", "")
	viper.SetDefault("auth.rate_limit", 5)
	viper.SetDefault("auth.lockout_minutes", 15)
	viper.SetDefault("site.title", "E4 Diary")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	Cfg = &Config{}
	if err := viper.Unmarshal(Cfg); err != nil {
		return err
	}

	if strings.EqualFold(Cfg.Server.Mode, "release") {
		if Cfg.Auth.Username == "admin" && Cfg.Auth.Password == defaultAdminPasswordHash {
			return errors.New("release 模式禁止使用默认管理员凭据，请通过环境变量或配置文件覆盖 auth.username 与 auth.password")
		}
		if Cfg.Auth.Secret == "your-secret-key-change-in-production" {
			return errors.New("release 模式必须修改 auth.secret")
		}
	}

	return nil
}
