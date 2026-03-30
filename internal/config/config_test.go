package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	value, exists := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if !exists {
			_ = os.Unsetenv(key)
			return
		}
		_ = os.Setenv(key, value)
	})
}

func TestLoadReadsPasswordFromConfigYAML(t *testing.T) {
	unsetEnvForTest(t, "E4_SERVER_HOST")
	unsetEnvForTest(t, "E4_SERVER_PORT")
	unsetEnvForTest(t, "E4_SERVER_MODE")
	unsetEnvForTest(t, "E4_AUTH_PASSWORD")
	unsetEnvForTest(t, "E4_AUTH_SECRET")
	unsetEnvForTest(t, "E4_AUTH_TOTP_SECRET")

	workdir := t.TempDir()
	configContent := "server:\n  mode: development\nauth:\n  username: admin\n  password: \"$2a$10$5OUxfHLfhWa1sYDlpuarQevoiPznWTmM1OZjLS.vtlbj7zsW6gMvG\"\nsite:\n  title: Test App\n"
	require.NoError(t, os.WriteFile(filepath.Join(workdir, "config.yaml"), []byte(configContent), 0o644))

	previousWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workdir))
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
		Cfg = nil
		sources = map[string]string{}
	})

	require.NoError(t, Load())
	assert.Equal(t, "$2a$10$5OUxfHLfhWa1sYDlpuarQevoiPznWTmM1OZjLS.vtlbj7zsW6gMvG", Cfg.Auth.Password)
	assert.Equal(t, "config.yaml", Source("auth.password"))
	assert.Equal(t, "admin", Cfg.Auth.Username)
	assert.Equal(t, "Test App", Cfg.Site.Title)
}

func TestLoadReadsSecretFromDotEnvWithoutDollarExpansion(t *testing.T) {
	unsetEnvForTest(t, "E4_SERVER_HOST")
	unsetEnvForTest(t, "E4_SERVER_PORT")
	unsetEnvForTest(t, "E4_SERVER_MODE")
	unsetEnvForTest(t, "E4_AUTH_SECRET")
	unsetEnvForTest(t, "E4_AUTH_TOTP_SECRET")
	unsetEnvForTest(t, "E4_AUTH_PASSWORD")

	workdir := t.TempDir()
	configContent := "auth:\n  password: \"$2a$10$5OUxfHLfhWa1sYDlpuarQevoiPznWTmM1OZjLS.vtlbj7zsW6gMvG\"\n  totp_secret: JBSWY3DPEHPK3PXP\n"
	dotEnvContent := "E4_SERVER_HOST=0.0.0.0\nE4_SERVER_PORT=9999\nE4_SERVER_MODE=release\nE4_AUTH_SECRET=$2a$10$secret-value\nE4_AUTH_TOTP_SECRET=SHOULD_NOT_LOAD_FROM_DOTENV\nE4_AUTH_PASSWORD=$2a$10$should-not-be-loaded-from-dotenv\n"
	require.NoError(t, os.WriteFile(filepath.Join(workdir, "config.yaml"), []byte(configContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(workdir, ".env"), []byte(dotEnvContent), 0o644))

	previousWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workdir))
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
		Cfg = nil
		sources = map[string]string{}
	})

	require.NoError(t, Load())
	assert.Equal(t, "0.0.0.0", Cfg.Server.Host)
	assert.Equal(t, ".env", Source("server.host"))
	assert.Equal(t, 9999, Cfg.Server.Port)
	assert.Equal(t, ".env", Source("server.port"))
	assert.Equal(t, "release", Cfg.Server.Mode)
	assert.Equal(t, ".env", Source("server.mode"))
	assert.Equal(t, "$2a$10$secret-value", Cfg.Auth.Secret)
	assert.Equal(t, ".env", Source("auth.secret"))
	assert.Equal(t, "JBSWY3DPEHPK3PXP", Cfg.Auth.TOTPSecret)
	assert.Equal(t, "config.yaml", Source("auth.totp_secret"))
	assert.Equal(t, "$2a$10$5OUxfHLfhWa1sYDlpuarQevoiPznWTmM1OZjLS.vtlbj7zsW6gMvG", Cfg.Auth.Password)
	assert.Equal(t, "config.yaml", Source("auth.password"))
	if value, ok := os.LookupEnv("E4_AUTH_PASSWORD"); ok {
		assert.NotEqual(t, "$2a$10$should-not-be-loaded-from-dotenv", value)
	}
	if value, ok := os.LookupEnv("E4_AUTH_TOTP_SECRET"); ok {
		assert.NotEqual(t, "SHOULD_NOT_LOAD_FROM_DOTENV", value)
	}
}

func TestLoadPrefersExplicitEnvironmentOverDotEnv(t *testing.T) {
	t.Setenv("E4_SERVER_PORT", "7777")
	t.Setenv("E4_AUTH_SECRET", "explicit-secret")
	unsetEnvForTest(t, "E4_AUTH_TOTP_SECRET")

	workdir := t.TempDir()
	configContent := "auth:\n  password: \"$2a$10$5OUxfHLfhWa1sYDlpuarQevoiPznWTmM1OZjLS.vtlbj7zsW6gMvG\"\n"
	dotEnvContent := "E4_SERVER_PORT=9999\nE4_AUTH_SECRET=dotenv-secret\n"
	require.NoError(t, os.WriteFile(filepath.Join(workdir, "config.yaml"), []byte(configContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(workdir, ".env"), []byte(dotEnvContent), 0o644))

	previousWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workdir))
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
		Cfg = nil
		sources = map[string]string{}
	})

	require.NoError(t, Load())
	assert.Equal(t, 7777, Cfg.Server.Port)
	assert.Equal(t, "env", Source("server.port"))
	assert.Equal(t, "explicit-secret", Cfg.Auth.Secret)
	assert.Equal(t, "env", Source("auth.secret"))
}
