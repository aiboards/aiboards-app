package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Environment  string `mapstructure:"ENVIRONMENT"`
	Port         int    `mapstructure:"PORT"`
	DatabaseURL  string `mapstructure:"DATABASE_URL"`
	JWTSecret    string `mapstructure:"JWT_SECRET"`
	CookieDomain string `mapstructure:"COOKIE_DOMAIN"`
	Version      string `mapstructure:"VERSION"`
	RateLimit    int    `mapstructure:"RATE_LIMIT"`

	// Admin User Configuration
	AdminEmail    string `mapstructure:"ADMIN_EMAIL"`
	AdminPassword string `mapstructure:"ADMIN_PASSWORD"`

	// JWT Configuration
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration

	// CORS Configuration
	AllowedOrigins []string `mapstructure:"ALLOWED_ORIGINS"`

	// Media Storage
	MediaStorageProvider string `mapstructure:"MEDIA_STORAGE_PROVIDER"`
	MediaStorageBucket   string `mapstructure:"MEDIA_STORAGE_BUCKET"`
	MediaStorageRegion   string `mapstructure:"MEDIA_STORAGE_REGION"`
	MediaStorageEndpoint string `mapstructure:"MEDIA_STORAGE_ENDPOINT"`
	MediaStorageKey      string `mapstructure:"MEDIA_STORAGE_KEY"`
	MediaStorageSecret   string `mapstructure:"MEDIA_STORAGE_SECRET"`
}

// LoadConfig loads the configuration from environment variables and config files
func LoadConfig(configPath string) (*Config, error) {
	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Set default values
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("ALLOWED_ORIGINS", []string{"http://localhost:3000"})
	viper.SetDefault("VERSION", "1.0.0")
	viper.SetDefault("RATE_LIMIT", 100) // 100 requests per minute per IP

	// Read environment variables
	viper.AutomaticEnv()
	_ = viper.BindEnv("DATABASE_URL")
	_ = viper.BindEnv("JWT_SECRET")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Set derived values
	config.AccessTokenDuration = 1 * time.Hour
	config.RefreshTokenDuration = 7 * 24 * time.Hour

	return &config, nil
}
