package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/config"
)

func TestLoad_WithRequiredEnvVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("SECRET_KEY", "test-secret-key")

	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseURL)
	assert.Equal(t, "test-secret-key", cfg.SecretKey)
	assert.Equal(t, "development", cfg.AppEnv) // default
	assert.Equal(t, 8080, cfg.Port)             // default
}


func TestIsProd(t *testing.T) {
	cfg := &config.Config{AppEnv: "production"}
	assert.True(t, cfg.IsProd())

	cfg.AppEnv = "development"
	assert.False(t, cfg.IsProd())

	cfg.AppEnv = "staging"
	assert.False(t, cfg.IsProd())
}

func TestOTPEnabled(t *testing.T) {
	t.Run("prod is always enabled", func(t *testing.T) {
		cfg := &config.Config{AppEnv: "production", OTPBypassCode: "test"}
		assert.True(t, cfg.OTPEnabled())
	})

	t.Run("dev with bypass code is disabled", func(t *testing.T) {
		cfg := &config.Config{AppEnv: "development", OTPBypassCode: "test"}
		assert.False(t, cfg.OTPEnabled())
	})

	t.Run("dev without bypass code is enabled", func(t *testing.T) {
		cfg := &config.Config{AppEnv: "development", OTPBypassCode: ""}
		assert.True(t, cfg.OTPEnabled())
	})
}

func TestCORSOriginsList(t *testing.T) {
	t.Run("returns default when empty", func(t *testing.T) {
		cfg := &config.Config{CORSOrigins: ""}
		origins := cfg.CORSOriginsList()
		assert.Equal(t, []string{"http://localhost:3000"}, origins)
	})

	t.Run("parses single origin", func(t *testing.T) {
		cfg := &config.Config{CORSOrigins: "https://example.com"}
		origins := cfg.CORSOriginsList()
		assert.Equal(t, []string{"https://example.com"}, origins)
	})

	t.Run("parses multiple origins", func(t *testing.T) {
		cfg := &config.Config{CORSOrigins: "https://example.com, https://another.com, https://third.com"}
		origins := cfg.CORSOriginsList()
		assert.Equal(t, []string{"https://example.com", "https://another.com", "https://third.com"}, origins)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		cfg := &config.Config{CORSOrigins: "  https://example.com  ,  https://another.com  "}
		origins := cfg.CORSOriginsList()
		assert.Equal(t, []string{"https://example.com", "https://another.com"}, origins)
	})

	t.Run("ignores empty entries", func(t *testing.T) {
		cfg := &config.Config{CORSOrigins: "https://example.com,,https://another.com"}
		origins := cfg.CORSOriginsList()
		assert.Equal(t, []string{"https://example.com", "https://another.com"}, origins)
	})
}
