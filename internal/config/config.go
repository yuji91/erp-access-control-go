package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Logger   LoggerConfig   `mapstructure:"logger"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret    string        `mapstructure:"secret"`
	ExpiresIn time.Duration `mapstructure:"expires_in"`
	Issuer    string        `mapstructure:"issuer"`
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load loads configuration from environment and config files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set default values
	setDefaults()

	// Read from environment variables
	viper.AutomaticEnv()

	// Bind environment variables with specific names
	bindEnvVariables()

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we'll use env vars and defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.name", "erp_access_control")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "password")
	viper.SetDefault("database.ssl_mode", "disable")

	// JWT defaults
	viper.SetDefault("jwt.secret", "your-super-secret-jwt-key-256-bits-long")
	viper.SetDefault("jwt.expires_in", "24h")
	viper.SetDefault("jwt.issuer", "erp-access-control-api")

	// Logger defaults
	viper.SetDefault("logger.level", "debug")
	viper.SetDefault("logger.format", "json")
}

// bindEnvVariables binds environment variables to configuration keys
func bindEnvVariables() {
	// Server
	viper.BindEnv("server.host", "SERVER_HOST")
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.mode", "GIN_MODE")

	// Database
	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.name", "DB_NAME")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("database.ssl_mode", "DB_SSL_MODE")

	// JWT
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	viper.BindEnv("jwt.expires_in", "JWT_EXPIRES_IN")
	viper.BindEnv("jwt.issuer", "JWT_ISSUER")

	// Logger
	viper.BindEnv("logger.level", "LOG_LEVEL")
	viper.BindEnv("logger.format", "LOG_FORMAT")
}

// GetDatabaseURL returns the database connection URL
func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetServerAddress returns the server address
func (c *Config) GetServerAddress() string {
	return c.Server.Host + ":" + c.Server.Port
}
