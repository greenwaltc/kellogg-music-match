package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration
type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	CORS         CORSConfig
	Artist       ArtistConfig
	Ticketmaster TicketmasterConfig
	Debug        DebugConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// CORSConfig holds CORS-related configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   string
	AllowedHeaders   string
	AllowCredentials bool
}

// ArtistConfig holds artist validation configuration
type ArtistConfig struct {
	MinCount        int
	MaxCount        int
	MaxNameLength   int
	SearchMaxLength int
	SearchLimit     int
}

// TicketmasterConfig holds Ticketmaster API configuration
type TicketmasterConfig struct {
	ConsumerKey    string
	ConsumerSecret string
	BaseURL        string
	Timeout        int // timeout in seconds
	MaxResults     int // maximum results per API call
	DefaultCity    string
	DefaultState   string
	DefaultCountry string
}

// DebugConfig holds debug-related configuration
type DebugConfig struct {
	Enabled bool
}

// Load creates a new Config instance from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnvWithDefault("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnvWithDefault("DB_HOST", "localhost"),
			Port:     getEnvWithDefault("DB_PORT", "5432"),
			Name:     getEnvWithDefault("DB_NAME", "kellogg_music_match"),
			User:     getEnvWithDefault("DB_USER", "kellogg_user"),
			Password: getEnvWithDefault("DB_PASSWORD", "kellogg_secure_pass_2024"),
			SSLMode:  getEnvWithDefault("DB_SSLMODE", "disable"),
		},
		CORS: CORSConfig{
			AllowedOrigins: strings.Split(
				getEnvWithDefault("CORS_ALLOWED_ORIGINS", "http://localhost:4200,http://kmm-ui.traefik.me"),
				",",
			),
			AllowedMethods:   getEnvWithDefault("CORS_ALLOWED_METHODS", "GET, POST, PUT, DELETE, OPTIONS"),
			AllowedHeaders:   getEnvWithDefault("CORS_ALLOWED_HEADERS", "Content-Type, Authorization, X-User-Username"),
			AllowCredentials: getEnvBoolWithDefault("CORS_ALLOW_CREDENTIALS", true),
		},
		Artist: ArtistConfig{
			MinCount:        getEnvIntWithDefault("ARTIST_MIN_COUNT", 5),
			MaxCount:        getEnvIntWithDefault("ARTIST_MAX_COUNT", 20),
			MaxNameLength:   getEnvIntWithDefault("ARTIST_MAX_NAME_LENGTH", 240),
			SearchMaxLength: getEnvIntWithDefault("ARTIST_SEARCH_MAX_LENGTH", 240),
			SearchLimit:     getEnvIntWithDefault("ARTIST_SEARCH_LIMIT", 10),
		},
		Ticketmaster: TicketmasterConfig{
			ConsumerKey:    getEnvWithDefault("TICKETMASTER_CONSUMER_KEY", ""),
			ConsumerSecret: getEnvWithDefault("TICKETMASTER_CONSUMER_SECRET", ""),
			BaseURL:        getEnvWithDefault("TICKETMASTER_BASE_URL", "https://app.ticketmaster.com/discovery/v2"),
			Timeout:        getEnvIntWithDefault("TICKETMASTER_TIMEOUT", 30),
			MaxResults:     getEnvIntWithDefault("TICKETMASTER_MAX_RESULTS", 200),
			DefaultCity:    getEnvWithDefault("TICKETMASTER_DEFAULT_CITY", "Chicago"),
			DefaultState:   getEnvWithDefault("TICKETMASTER_DEFAULT_STATE", "IL"),
			DefaultCountry: getEnvWithDefault("TICKETMASTER_DEFAULT_COUNTRY", "US"),
		},
		Debug: DebugConfig{
			Enabled: getEnvBoolWithDefault("DEBUG_ENABLED", false),
		},
	}
}

// ConnectionString returns the PostgreSQL connection string
func (c *DatabaseConfig) ConnectionString() string {
	return "host=" + c.Host + " port=" + c.Port + " user=" + c.User +
		" password=" + c.Password + " dbname=" + c.Name + " sslmode=" + c.SSLMode
}

// getEnvWithDefault returns the value of the environment variable or a default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntWithDefault returns the integer value of the environment variable or a default value
func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBoolWithDefault returns the boolean value of the environment variable or a default value
func getEnvBoolWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
