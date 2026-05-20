package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// App
	AppEnv  string `envconfig:"APP_ENV" default:"development"`
	Port    int    `envconfig:"PORT" default:"8080"`
	FrontendURL string `envconfig:"FRONTEND_URL" default:"http://localhost:3000"`
	CORSOrigins string `envconfig:"CORS_ORIGINS" default:"http://localhost:3000"`

	// Database
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	// JWT & Security
	SecretKey                  string `envconfig:"SECRET_KEY" required:"true"`
	AccessTokenExpireMinutes   int    `envconfig:"ACCESS_TOKEN_EXPIRE_MINUTES" default:"15"`

	// OTP
	OTPBypassCode   string `envconfig:"OTP_BYPASS_CODE"`
	TwilioAccountSID string `envconfig:"TWILIO_ACCOUNT_SID"`
	TwilioAuthToken  string `envconfig:"TWILIO_AUTH_TOKEN"`
	TwilioVerifySID  string `envconfig:"TWILIO_VERIFY_SID"`

	// Supabase Storage
	SupabaseURL            string `envconfig:"SUPABASE_URL"`
	SupabaseServiceRoleKey string `envconfig:"SUPABASE_SERVICE_ROLE_KEY"`

	// Anthropic
	AnthropicAPIKey string `envconfig:"ANTHROPIC_API_KEY"`
	LLMModel        string `envconfig:"LLM_MODEL" default:"claude-haiku-4-5"`
	ChatRateLimit   int    `envconfig:"CHAT_RATE_LIMIT" default:"20"`

	// Stripe
	StripeSecretKey       string `envconfig:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret   string `envconfig:"STRIPE_WEBHOOK_SECRET"`
	StripePriceBasicMonthly string `envconfig:"STRIPE_PRICE_BASIC_MONTHLY"`
	StripePriceBasicYearly  string `envconfig:"STRIPE_PRICE_BASIC_YEARLY"`
	StripePriceProMonthly   string `envconfig:"STRIPE_PRICE_PRO_MONTHLY"`
	StripePriceProYearly    string `envconfig:"STRIPE_PRICE_PRO_YEARLY"`

	// VAPID
	VAPIDPrivateKey   string `envconfig:"VAPID_PRIVATE_KEY"`
	VAPIDPublicKey    string `envconfig:"VAPID_PUBLIC_KEY"`
	VAPIDClaimsEmail  string `envconfig:"VAPID_CLAIMS_EMAIL" default:"admin@rachao.app"`
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env") // silently ignored if file doesn't exist
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &c, nil
}

func (c *Config) IsProd() bool {
	return c.AppEnv == "production"
}

func (c *Config) OTPEnabled() bool {
	return c.OTPBypassCode == "" || c.IsProd()
}

func (c *Config) CORSOriginsList() []string {
	if c.CORSOrigins == "" {
		return []string{"http://localhost:3000"}
	}
	var origins []string
	for _, o := range strings.Split(c.CORSOrigins, ",") {
		if s := strings.TrimSpace(o); s != "" {
			origins = append(origins, s)
		}
	}
	return origins
}
