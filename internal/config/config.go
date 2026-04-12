package config

import (
	"fmt"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type AppConfig struct {
	GRPCPort    string `mapstructure:"GRPC_PORT" validate:"required"`
	HTTPPort    string `mapstructure:"HTTP_PORT" validate:"required"`
	Environment string `mapstructure:"ENVIRONMENT" validate:"required,oneof=development production staging"`

	DatabaseURL string `mapstructure:"DATABASE_URL" validate:"required,url"`

	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY" validate:"required,len=32"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION" validate:"required"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION" validate:"required"`

	MemcachedURL string `mapstructure:"MEMCACHED_URL" validate:"required"`
	RedisURL     string `mapstructure:"REDIS_URL" validate:"required"`

	GoogleClientID     string `mapstructure:"GOOGLE_CLIENT_ID" validate:"required"`
	GoogleClientSecret string `mapstructure:"GOOGLE_CLIENT_SECRET" validate:"required"`
	GoogleCallbackURL  string `mapstructure:"GOOGLE_CALLBACK_URL" validate:"required,url"`

	LogLevel string `mapstructure:"LOG_LEVEL" validate:"required"`

	// SMTP
	SMTPHost     string `mapstructure:"SMTP_HOST" validate:"required"`
	SMTPPort     int    `mapstructure:"SMTP_PORT" validate:"required,gt=0"`
	SMTPUsername string `mapstructure:"SMTP_USERNAME"`
	SMTPPassword string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom     string `mapstructure:"SMTP_FROM" validate:"required"`

	// Scheduler
	ReminderCron string `mapstructure:"REMINDER_CRON" validate:"required"`
	AppURL       string `mapstructure:"APP_URL" validate:"required,url"`
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

	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			fmt.Printf("Config error: Field '%s' failed on the '%s' tag\n", err.Field(), err.Tag())
		}
		log.Fatalf("Config validation failed")
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
