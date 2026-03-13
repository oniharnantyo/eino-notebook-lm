package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server  ServerConfig
	Database DatabaseConfig
	Gemini  GeminiConfig
	Log     LogConfig
	Cache   CacheConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver          string
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string
	Format string // json, text
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled bool
	Host    string
	Port    int
	Prefix  string
	TTL     time.Duration
}

// GeminiConfig holds Gemini API configuration for embeddings
type GeminiConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	Dimension  int
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	// Also support EINO_ prefixed env vars for backward compatibility
	viper.SetEnvPrefix("EINO")
	viper.BindEnv("server.host", "SERVER_HOST")
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("log.level", "LOG_LEVEL")
	viper.BindEnv("database.driver", "DATABASE_DRIVER")
	viper.BindEnv("database.host", "DATABASE_HOST")
	viper.BindEnv("database.port", "DATABASE_PORT")
	viper.BindEnv("database.user", "DATABASE_USER")
	viper.BindEnv("database.password", "DATABASE_PASSWORD")
	viper.BindEnv("database.database", "DATABASE_NAME")
	viper.BindEnv("gemini.api_key", "GEMINI_API_KEY")
	viper.BindEnv("gemini.base_url", "GEMINI_BASE_URL")
	viper.BindEnv("gemini.model", "GEMINI_MODEL")
	viper.BindEnv("gemini.dimension", "GEMINI_DIMENSION")

	// Set defaults
	setDefaults()

	cfg := &Config{}

	// Unmarshal config
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 15*time.Second)
	viper.SetDefault("server.write_timeout", 15*time.Second)
	viper.SetDefault("server.idle_timeout", 60*time.Second)

	// Database defaults
	viper.SetDefault("database.driver", "postgres")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")

	// Cache defaults
	viper.SetDefault("cache.enabled", false)
	viper.SetDefault("cache.port", 6379)
	viper.SetDefault("cache.ttl", 5*time.Minute)

	// Gemini defaults
	viper.SetDefault("gemini.base_url", "https://generativelanguage.googleapis.com")
	viper.SetDefault("gemini.model", "text-embedding-004")
	viper.SetDefault("gemini.dimension", 768)
}

// GetServerAddress returns the server address
func (c *ServerConfig) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=disable",
		c.Driver, c.User, c.Password, c.Host, c.Port, c.Database)
}
