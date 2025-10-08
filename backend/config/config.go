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
	Matching     MatchingConfig
	Ticketmaster TicketmasterConfig
	Debug        DebugConfig
	Telemetry    TelemetryConfig
	JWT          JWTConfig
	Email        EmailConfig
	Spotify      SpotifyConfig
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

// MatchingConfig holds matching service defaults
type MatchingConfig struct {
	DefaultRange  string
	DefaultLimit  int
	MaxLimit      int
	AllowedRanges []string
	MaxOverlaps   int // safety cap for overlapsLimit (0 = no cap)
}

// TicketmasterConfig holds Ticketmaster API configuration
type TicketmasterConfig struct {
	ConsumerKey     string
	ConsumerSecret  string
	BaseURL         string
	Timeout         int // timeout in seconds
	MaxResults      int // maximum results per API call
	DefaultCity     string
	DefaultState    string
	DefaultCountry  string
	DateRangeMonths int // number of months to look ahead for events
	// Optional geo-based search (overrides city/state when provided)
	GeoLatLong string // e.g. "41.8781,-87.6298"
	Radius     int    // distance from geo point
	RadiusUnit string // miles (default) or km
}

// DebugConfig holds debug-related configuration
type DebugConfig struct {
	Enabled bool
}

// TelemetryConfig controls tracing/metrics exporters
type TelemetryConfig struct {
	Enabled        bool   // master toggle
	Exporter       string // "stdout" (default) or "otlp"
	OTLPEndpoint   string // e.g. http://otel-collector:4318
	ServiceName    string // override service name
	ServiceVersion string // override service version
}

// EmailConfig holds email service configuration
type EmailConfig struct {
	Provider  string // "sendgrid", "ses", "smtp"
	APIKey    string // For SendGrid
	FromEmail string
	FromName  string
	SMTPHost  string // For SMTP
	SMTPPort  string
	SMTPUser  string
	SMTPPass  string
	Enabled   bool
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	SecretKey     string
	ExpiryHours   int
	RefreshHours  int
	LeewaySeconds int // allowed clock skew (nbf/iat/exp) in seconds
}

// SpotifyConfig holds Spotify API configuration
type SpotifyConfig struct {
	ClientID        string
	ClientSecret    string
	RefreshTokenKey string
	RedirectURI     string
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
		Matching: MatchingConfig{
			DefaultRange:  getEnvWithDefault("MATCHING_DEFAULT_RANGE", "medium_term"),
			DefaultLimit:  getEnvIntWithDefault("MATCHING_DEFAULT_LIMIT", 10),
			MaxLimit:      getEnvIntWithDefault("MATCHING_MAX_LIMIT", 50),
			AllowedRanges: []string{"short_term", "medium_term", "long_term"},
			MaxOverlaps:   getEnvIntWithDefault("MATCHING_MAX_OVERLAPS", 100),
		},
		Ticketmaster: TicketmasterConfig{
			ConsumerKey:     getEnvWithDefault("TICKETMASTER_CONSUMER_KEY", ""),
			ConsumerSecret:  getEnvWithDefault("TICKETMASTER_CONSUMER_SECRET", ""),
			BaseURL:         getEnvWithDefault("TICKETMASTER_BASE_URL", "https://app.ticketmaster.com/discovery/v2"),
			Timeout:         getEnvIntWithDefault("TICKETMASTER_TIMEOUT", 30),
			MaxResults:      getEnvIntWithDefault("TICKETMASTER_MAX_RESULTS", 200),
			DefaultCity:     getEnvWithDefault("TICKETMASTER_DEFAULT_CITY", "Chicago"),
			DefaultState:    getEnvWithDefault("TICKETMASTER_DEFAULT_STATE", "IL"),
			DefaultCountry:  getEnvWithDefault("TICKETMASTER_DEFAULT_COUNTRY", "US"),
			DateRangeMonths: getEnvIntWithDefault("TICKETMASTER_DATE_RANGE_MONTHS", 6),
			GeoLatLong:      getEnvWithDefault("TICKETMASTER_GEO_LATLONG", ""),
			Radius:          getEnvIntWithDefault("TICKETMASTER_RADIUS", 0),
			RadiusUnit:      getEnvWithDefault("TICKETMASTER_RADIUS_UNIT", "miles"),
		},
		Debug: DebugConfig{
			Enabled: getEnvBoolWithDefault("DEBUG_ENABLED", false),
		},
		Telemetry: TelemetryConfig{
			Enabled:        getEnvBoolWithDefault("TRACING_ENABLED", true),
			Exporter:       getEnvWithDefault("TRACING_EXPORTER", "stdout"),
			OTLPEndpoint:   getEnvWithDefault("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			ServiceName:    getEnvWithDefault("OTEL_SERVICE_NAME", "kmm-backend"),
			ServiceVersion: getEnvWithDefault("OTEL_SERVICE_VERSION", "1.0.0"),
		},
		JWT: JWTConfig{
			SecretKey:     getEnvWithDefault("JWT_SECRET_KEY", "your-secret-key-change-in-production"),
			ExpiryHours:   getEnvIntWithDefault("JWT_EXPIRY_HOURS", 24),
			RefreshHours:  getEnvIntWithDefault("JWT_REFRESH_HOURS", 168), // 7 days
			LeewaySeconds: getEnvIntWithDefault("JWT_LEEWAY_SECONDS", 120),
		},
		Email: EmailConfig{
			Provider:  getEnvWithDefault("EMAIL_PROVIDER", "sendgrid"),
			APIKey:    getEnvWithDefault("SENDGRID_API_KEY", ""),
			FromEmail: getEnvWithDefault("EMAIL_FROM_EMAIL", "noreply@kellogg-music-match.com"),
			FromName:  getEnvWithDefault("EMAIL_FROM_NAME", "Kellogg Music Match"),
			SMTPHost:  getEnvWithDefault("SMTP_HOST", ""),
			SMTPPort:  getEnvWithDefault("SMTP_PORT", "587"),
			SMTPUser:  getEnvWithDefault("SMTP_USER", ""),
			SMTPPass:  getEnvWithDefault("SMTP_PASS", ""),
			Enabled:   getEnvBoolWithDefault("EMAIL_ENABLED", false),
		},
		Spotify: SpotifyConfig{
			ClientID:        getEnvWithDefault("SPOTIFY_CLIENT_ID", "spotify-client-id"),
			ClientSecret:    getEnvWithDefault("SPOTIFY_CLIENT_SECRET", "spotify-client-secret"),
			RefreshTokenKey: getEnvWithDefault("SPOTIFY_REFRESH_TOKEN_KEY", ""),
			RedirectURI:     getEnvWithDefault("SPOTIFY_REDIRECT_URI", "http://localhost:4200/spotify/callback"),
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
