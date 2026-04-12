package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type AppConfig struct {
	GRPCPort    string `mapstructure:"GRPC_PORT"`
	HTTPPort    string `mapstructure:"HTTP_PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`

	DatabaseURL string `mapstructure:"DATABASE_URL"`

	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`

	MemcachedURL string `mapstructure:"MEMCACHED_URL"`
	RedisURL     string `mapstructure:"REDIS_URL"`

	GoogleClientID     string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	GoogleCallbackURL  string `mapstructure:"GOOGLE_CALLBACK_URL"`

	LogLevel string `mapstructure:"LOG_LEVEL"`

	// SMTP
	SMTPHost     string `mapstructure:"SMTP_HOST"`
	SMTPPort     int    `mapstructure:"SMTP_PORT"`
	SMTPUsername string `mapstructure:"SMTP_USERNAME"`
	SMTPPassword string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom     string `mapstructure:"SMTP_FROM"`

	// Scheduler
	ReminderCron string `mapstructure:"REMINDER_CRON"`
	AppURL       string `mapstructure:"APP_URL"`
}

func Load() *AppConfig {
	viper.SetConfigFile(".env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("GRPC_PORT", "9090")
	viper.SetDefault("HTTP_PORT", "8080")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ACCESS_TOKEN_DURATION", "15m")
	viper.SetDefault("REFRESH_TOKEN_DURATION", "24h")
	viper.SetDefault("MEMCACHED_URL", "localhost:11211")
	viper.SetDefault("REDIS_URL", "localhost:6379")

	// SMTP defaults (Mailpit for local dev)
	viper.SetDefault("SMTP_HOST", "localhost")
	viper.SetDefault("SMTP_PORT", 1025)
	viper.SetDefault("SMTP_FROM", "noreply@todo-app.local")

	// Scheduler defaults
	viper.SetDefault("REMINDER_CRON", "0 * * * * *") // every minute
	viper.SetDefault("APP_URL", "http://localhost:8080")

	_ = viper.ReadInConfig()

	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("unmarshal config: %v", err)
	}
	return &cfg
}

func (c *AppConfig) IsDevelopment() bool { return c.Environment == "development" }
func (c *AppConfig) IsProduction() bool  { return c.Environment == "production" }

func (c *AppConfig) SafeLog() map[string]any {
	return map[string]any{
		"grpc_port":   c.GRPCPort,
		"http_port":   c.HTTPPort,
		"environment": c.Environment,
		"log_level":   c.LogLevel,
	}
}
