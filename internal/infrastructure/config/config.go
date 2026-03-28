package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Gemini     GeminiConfig
	OpenAI     OpenAIConfig
	Model      ModelConfig
	Log        LogConfig
	Cache      CacheConfig
	Kreuzberg  KreuzbergConfig
	Langfuse   LangfuseConfig
	Transformer TransformerConfig
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
	Driver          string `validate:"required,oneof=postgres"`
	Host            string `validate:"required,hostname|ip"`
	Port            int    `validate:"required,min=1,max=65535"`
	User            string `validate:"required"`
	Password        string
	Database        string `validate:"required"`
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

// GeminiConfig holds Gemini API configuration
type GeminiConfig struct {
	APIKey  string `mapstructure:"api_key" validate:"required"`
	BaseURL string `mapstructure:"base_url" validate:"required,url"`
	// Deprecated: Use ModelConfig instead
	EmbeddingModel string `mapstructure:"embedding_model"`
	ChatModel      string `mapstructure:"chat_model"`
	Dimension      int     `mapstructure:"dimension"`
}

// OpenAIConfig holds OpenAI API configuration
type OpenAIConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// ModelConfig holds model configuration with provider prefix support
// Format: "provider/model-name" (e.g., "gemini/gemini-2.0-flash-exp", "openai/gpt-4o-mini")
type ModelConfig struct {
	ChatModel          string `mapstructure:"chat_model" validate:"required"`
	EmbeddingModel     string `mapstructure:"embedding_model" validate:"required"`
	EmbeddingDimension int    `mapstructure:"embedding_dimension" validate:"required,min=1"`
}

// KreuzbergConfig holds Kreuzberg document extractor configuration
type KreuzbergConfig struct {
	ServiceURL   string        `mapstructure:"service_url" validate:"required,url"`
	OutputFormat string        `mapstructure:"output_format" validate:"omitempty,oneof=markdown text html"`
	Timeout      time.Duration `mapstructure:"timeout"`
}

// LangfuseConfig holds Langfuse observability configuration
type LangfuseConfig struct {
	Host       string  `mapstructure:"host" validate:"omitempty,url"`
	PublicKey  string  `mapstructure:"public_key"`
	SecretKey  string  `mapstructure:"secret_key"`
	Enabled    bool    `mapstructure:"enabled"`
	SampleRate float64 `mapstructure:"sample_rate" validate:"omitempty,min=0,max=1"`
	Release    string  `mapstructure:"release"`
}

// TransformerConfig holds document transformer configuration
type TransformerConfig struct {
	Type                    string                   `mapstructure:"type" validate:"required,oneof=recursive markdown"`
	RecursiveSplitterConfig *RecursiveSplitterConfig `mapstructure:"recursive_splitter"`
}

// RecursiveSplitterConfig holds recursive chunk splitter configuration
type RecursiveSplitterConfig struct {
	ChunkSize   int `mapstructure:"chunk_size" validate:"required,min=1"`
	OverlapSize int `mapstructure:"overlap_size" validate:"required,min=0"`
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.BindEnv("server.host", "SERVER_HOST")
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	viper.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	viper.BindEnv("server.idle_timeout", "SERVER_IDLE_TIMEOUT")
	viper.BindEnv("log.level", "LOG_LEVEL")
	viper.BindEnv("log.format", "LOG_FORMAT")
	viper.BindEnv("database.driver", "DATABASE_DRIVER")
	viper.BindEnv("database.host", "DATABASE_HOST")
	viper.BindEnv("database.port", "DATABASE_PORT")
	viper.BindEnv("database.user", "DATABASE_USER")
	viper.BindEnv("database.password", "DATABASE_PASSWORD")
	viper.BindEnv("database.database", "DATABASE_NAME")
	viper.BindEnv("gemini.api_key", "GEMINI_API_KEY")
	viper.BindEnv("gemini.base_url", "GEMINI_BASE_URL")
	viper.BindEnv("gemini.embedding_model", "GEMINI_EMBEDDING_MODEL")
	viper.BindEnv("gemini.chat_model", "GEMINI_CHAT_MODEL")
	viper.BindEnv("gemini.dimension", "GEMINI_DIMENSION")
	viper.BindEnv("openai.api_key", "OPENAI_API_KEY")
	viper.BindEnv("openai.base_url", "OPENAI_BASE_URL")
	viper.BindEnv("model.chat_model", "CHAT_MODEL")
	viper.BindEnv("model.embedding_model", "EMBEDDING_MODEL")
	viper.BindEnv("model.embedding_dimension", "EMBEDDING_DIMENSION")
	viper.BindEnv("kreuzberg.service_url", "KREUZBERG_SERVICE_URL")
	viper.BindEnv("kreuzberg.output_format", "KREUZBERG_OUTPUT_FORMAT")
	viper.BindEnv("kreuzberg.timeout", "KREUZBERG_TIMEOUT")
	viper.BindEnv("kreuzberg.to_pages", "KREUZBERG_TO_PAGES")
	viper.BindEnv("langfuse.host", "LANGFUSE_HOST")
	viper.BindEnv("langfuse.public_key", "LANGFUSE_PUBLIC_KEY")
	viper.BindEnv("langfuse.secret_key", "LANGFUSE_SECRET_KEY")
	viper.BindEnv("langfuse.enabled", "LANGFUSE_ENABLED")
	viper.BindEnv("langfuse.sample_rate", "LANGFUSE_SAMPLE_RATE")
	viper.BindEnv("langfuse.release", "LANGFUSE_RELEASE")
	viper.BindEnv("transformer.type", "TRANSFORMER_TYPE")
	viper.BindEnv("transformer.recursive_splitter.chunk_size", "TRANSFORMER_CHUNK_SIZE")
	viper.BindEnv("transformer.recursive_splitter.overlap_size", "TRANSFORMER_OVERLAP_SIZE")

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
	// Note: APIKey, Model, and Dimension are required - no defaults to catch config errors
	viper.SetDefault("gemini.base_url", "https://generativelanguage.googleapis.com")

	// Model defaults (multi-provider with provider/model format)
	viper.SetDefault("model.chat_model", "gemini/gemini-2.0-flash-exp")
	viper.SetDefault("model.embedding_model", "gemini/text-embedding-004")
	viper.SetDefault("model.embedding_dimension", 768)

	// Kreuzberg defaults
	viper.SetDefault("kreuzberg.service_url", "http://localhost:8000")
	viper.SetDefault("kreuzberg.output_format", "markdown")
	viper.SetDefault("kreuzberg.timeout", 30*time.Second)

	// Langfuse defaults
	viper.SetDefault("langfuse.host", "https://cloud.langfuse.com")
	viper.SetDefault("langfuse.enabled", false)
	viper.SetDefault("langfuse.sample_rate", 1.0)

	// Transformer defaults
	viper.SetDefault("transformer.type", "recursive")
	viper.SetDefault("transformer.recursive_splitter.chunk_size", 4000)
	viper.SetDefault("transformer.recursive_splitter.overlap_size", 800)
}

// GetServerAddress returns the server address
func (c *ServerConfig) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Validate validates the configuration and returns an error if any required field is missing
func (c *Config) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Custom validation: overlap_size must be less than chunk_size
	if c.Transformer.RecursiveSplitterConfig != nil {
		if c.Transformer.RecursiveSplitterConfig.OverlapSize >= c.Transformer.RecursiveSplitterConfig.ChunkSize {
			return fmt.Errorf("config validation failed: TRANSFORMER_OVERLAP_SIZE (%d) must be less than TRANSFORMER_CHUNK_SIZE (%d)",
				c.Transformer.RecursiveSplitterConfig.OverlapSize,
				c.Transformer.RecursiveSplitterConfig.ChunkSize)
		}
	}

	return nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=disable",
		c.Driver, c.User, c.Password, c.Host, c.Port, c.Database)
}

