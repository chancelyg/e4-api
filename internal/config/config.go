package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
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

var sources = map[string]string{}

type envBinding struct {
	key         string
	env         string
	allowDotEnv bool
}

var envBindings = []envBinding{
	{key: "server.host", env: "E4_SERVER_HOST", allowDotEnv: true},
	{key: "server.port", env: "E4_SERVER_PORT", allowDotEnv: true},
	{key: "server.mode", env: "E4_SERVER_MODE", allowDotEnv: true},
	{key: "database.dsn", env: "E4_DATABASE_DSN"},
	{key: "auth.username", env: "E4_AUTH_USERNAME"},
	{key: "auth.password", env: "E4_AUTH_PASSWORD"},
	{key: "auth.secret", env: "E4_AUTH_SECRET", allowDotEnv: true},
	{key: "auth.totp_secret", env: "E4_AUTH_TOTP_SECRET"},
	{key: "auth.rate_limit", env: "E4_AUTH_RATE_LIMIT"},
	{key: "auth.lockout_minutes", env: "E4_AUTH_LOCKOUT_MINUTES"},
	{key: "site.title", env: "E4_SITE_TITLE"},
}

func Load() error {
	existingEnv := captureExistingEnv(envBindings)
	dotEnvEnv, err := loadDotEnvIfPresent(envBindings)
	if err != nil {
		return err
	}

	v := viper.New()

	// Bind environment variables
	if err := bindEnv(v, envBindings...); err != nil {
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

	sources = map[string]string{
		"server.host":          resolveSource(v, existingEnv, dotEnvEnv, "server.host", "E4_SERVER_HOST"),
		"server.port":          resolveSource(v, existingEnv, dotEnvEnv, "server.port", "E4_SERVER_PORT"),
		"server.mode":          resolveSource(v, existingEnv, dotEnvEnv, "server.mode", "E4_SERVER_MODE"),
		"database.dsn":         resolveSource(v, existingEnv, dotEnvEnv, "database.dsn", "E4_DATABASE_DSN"),
		"auth.username":        resolveSource(v, existingEnv, dotEnvEnv, "auth.username", "E4_AUTH_USERNAME"),
		"auth.password":        resolveSource(v, existingEnv, dotEnvEnv, "auth.password", "E4_AUTH_PASSWORD"),
		"auth.secret":          resolveSource(v, existingEnv, dotEnvEnv, "auth.secret", "E4_AUTH_SECRET"),
		"auth.totp_secret":     resolveSource(v, existingEnv, dotEnvEnv, "auth.totp_secret", "E4_AUTH_TOTP_SECRET"),
		"auth.rate_limit":      resolveSource(v, existingEnv, dotEnvEnv, "auth.rate_limit", "E4_AUTH_RATE_LIMIT"),
		"auth.lockout_minutes": resolveSource(v, existingEnv, dotEnvEnv, "auth.lockout_minutes", "E4_AUTH_LOCKOUT_MINUTES"),
		"site.title":           resolveSource(v, existingEnv, dotEnvEnv, "site.title", "E4_SITE_TITLE"),
	}

	Cfg.Server.Host = strings.TrimSpace(Cfg.Server.Host)
	Cfg.Server.Mode = strings.TrimSpace(Cfg.Server.Mode)
	Cfg.Auth.Username = strings.TrimSpace(Cfg.Auth.Username)
	Cfg.Auth.Password = strings.TrimSpace(Cfg.Auth.Password)
	Cfg.Auth.Secret = strings.TrimSpace(Cfg.Auth.Secret)
	Cfg.Auth.TOTPSecret = strings.TrimSpace(Cfg.Auth.TOTPSecret)
	Cfg.Site.Title = strings.TrimSpace(Cfg.Site.Title)

	if strings.EqualFold(Cfg.Server.Mode, "release") {
		if Cfg.Auth.Username == "" {
			return errors.New("release 模式必须设置非空的 auth.username")
		}
		if Cfg.Auth.Password == defaultAdminPasswordHash {
			return errors.New("release 模式禁止使用默认管理员密码哈希，请通过 config.yaml 覆盖 auth.password")
		}
		if Cfg.Auth.Secret == "" || Cfg.Auth.Secret == defaultDevAuthSecret {
			return errors.New("release 模式必须修改 auth.secret")
		}
	}

	return nil
}

func loadDotEnvIfPresent(bindings []envBinding) (map[string]struct{}, error) {
	loaded := make(map[string]struct{})
	allowed := make(map[string]struct{})
	for _, binding := range bindings {
		if binding.allowDotEnv {
			allowed[binding.env] = struct{}{}
		}
	}

	if _, err := os.Stat(".env"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return loaded, nil
		}
		return nil, err
	}

	file, err := os.Open(".env")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		key, value, ok, err := parseDotEnvLine(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("parse .env line %d: %w", lineNo, err)
		}
		if !ok {
			continue
		}
		if _, permitted := allowed[key]; !permitted {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return nil, err
		}
		loaded[key] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return loaded, nil
}

func bindEnv(v *viper.Viper, bindings ...envBinding) error {
	for _, binding := range bindings {
		if err := v.BindEnv(binding.key, binding.env); err != nil {
			return err
		}
	}
	return nil
}

func Source(key string) string {
	if source, ok := sources[key]; ok {
		return source
	}
	return "unknown"
}

func UsesDefaultAdminPasswordHash() bool {
	return Cfg != nil && Cfg.Auth.Password == defaultAdminPasswordHash
}

func UsesDefaultDevAuthSecret() bool {
	return Cfg != nil && Cfg.Auth.Secret == defaultDevAuthSecret
}

func captureExistingEnv(bindings []envBinding) map[string]struct{} {
	existing := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		if _, ok := os.LookupEnv(binding.env); ok {
			existing[binding.env] = struct{}{}
		}
	}
	return existing
}

func resolveSource(v *viper.Viper, existingEnv, dotEnvEnv map[string]struct{}, key, envName string) string {
	if envName != "" {
		if _, ok := existingEnv[envName]; ok {
			return "env"
		}
		if _, ok := dotEnvEnv[envName]; ok {
			return ".env"
		}
	}
	if v.InConfig(key) {
		return "config.yaml"
	}
	return "default"
}

func parseDotEnvLine(line string) (key, value string, ok bool, err error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false, nil
	}

	trimmed = strings.TrimPrefix(trimmed, "export ")
	separator := strings.Index(trimmed, "=")
	if separator <= 0 {
		return "", "", false, fmt.Errorf("invalid assignment")
	}

	key = strings.TrimSpace(trimmed[:separator])
	if key == "" {
		return "", "", false, fmt.Errorf("empty key")
	}

	value, err = parseDotEnvValue(strings.TrimSpace(trimmed[separator+1:]))
	if err != nil {
		return "", "", false, err
	}

	return key, value, true, nil
}

func parseDotEnvValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	if raw[0] == '\'' || raw[0] == '"' {
		quote := raw[0]
		if len(raw) < 2 {
			return "", fmt.Errorf("unterminated quoted value")
		}
		end := -1
		escaped := false
		for i := 1; i < len(raw); i++ {
			if quote == '"' && raw[i] == '\\' && !escaped {
				escaped = true
				continue
			}
			if raw[i] == quote && !escaped {
				end = i
				break
			}
			escaped = false
		}
		if end == -1 {
			return "", fmt.Errorf("unterminated quoted value")
		}

		trailing := strings.TrimSpace(raw[end+1:])
		if trailing != "" && !strings.HasPrefix(trailing, "#") {
			return "", fmt.Errorf("unexpected trailing content")
		}

		value := raw[1:end]
		if quote == '"' {
			replacer := strings.NewReplacer(`\\`, `\`, `\n`, "\n", `\r`, "\r", `\t`, "\t", `\"`, `"`)
			value = replacer.Replace(value)
		}
		return value, nil
	}

	if comment := strings.Index(raw, " #"); comment >= 0 {
		raw = raw[:comment]
	}

	return strings.TrimSpace(raw), nil
}
