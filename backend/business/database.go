package business

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
)

// DatabaseConfig holds configuration for database connection
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// NewDatabaseConfigFromEnv creates a database config from environment variables
func NewDatabaseConfigFromEnv() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnvWithDefault("DB_HOST", "localhost"),
		Port:     getEnvWithDefault("DB_PORT", "5432"),
		Name:     getEnvWithDefault("DB_NAME", "kellogg_music_match"),
		User:     getEnvWithDefault("DB_USER", "kellogg_user"),
		Password: getEnvWithDefault("DB_PASSWORD", "kellogg_secure_pass_2024"),
		SSLMode:  getEnvWithDefault("DB_SSLMODE", "disable"),
	}
}

// ConnectionString returns the PostgreSQL connection string
func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// User operations
	CreateUser(ctx context.Context, id uuid.UUID, username, email, firstName, lastName, passwordHash string) (*sqlc.User, error)
	GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error)
	GetUserByUsernameWithPassword(ctx context.Context, username string) (*sqlc.User, error)
	GetUserByEmail(ctx context.Context, email string) (*sqlc.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*sqlc.User, error)
	UserExistsByUsername(ctx context.Context, username string) (bool, error)
	UserExistsByEmail(ctx context.Context, email string) (bool, error)
	GetAllUsersWithArtists(ctx context.Context) ([]sqlc.GetUsersWithArtistsRow, error)

	// Artist operations
	CreateArtist(ctx context.Context, name string) (*sqlc.Artist, error)
	GetArtistByName(ctx context.Context, name string) (*sqlc.Artist, error)
	SearchArtists(ctx context.Context, query string, limit int32) ([]sqlc.Artist, error)

	// User-Artist relationship operations
	SetUserArtists(ctx context.Context, userID uuid.UUID, artistNames []string) error
	GetUserArtists(ctx context.Context, userID uuid.UUID) ([]sqlc.Artist, error)
	ClearUserArtists(ctx context.Context, userID uuid.UUID) error

	// Matching operations
	FindSimilarUsers(ctx context.Context, username string) ([]sqlc.FindSimilarUsersRow, error)

	// Feedback operations
	CreateFeedback(ctx context.Context, userID uuid.UUID, feedbackText string) (*sqlc.Feedback, error)
	GetFeedbackByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Feedback, error)
} // PostgreSQLUserRepository implements UserRepository using PostgreSQL
type PostgreSQLUserRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
}

// NewPostgreSQLUserRepository creates a new PostgreSQL user repository
func NewPostgreSQLUserRepository(config *DatabaseConfig) (*PostgreSQLUserRepository, error) {
	db, err := sql.Open("postgres", config.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgreSQLUserRepository{
		db:      db,
		queries: sqlc.New(db),
	}, nil
}

// NewUserRepository creates a new UserRepository with default database configuration
func NewUserRepository() (UserRepository, error) {
	config := NewDatabaseConfigFromEnv()

	// Log connection attempt (without password)
	fmt.Printf("Attempting database connection to %s:%s@%s:%s/%s\n",
		config.User, "***", config.Host, config.Port, config.Name)

	repo, err := NewPostgreSQLUserRepository(config)
	if err != nil {
		fmt.Printf("Database connection failed: %v\n", err)
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}

	fmt.Println("Database connection successful!")
	return repo, nil
}

// Close closes the database connection
func (r *PostgreSQLUserRepository) Close() error {
	return r.db.Close()
}

// CreateUser creates a new user in the database
func (r *PostgreSQLUserRepository) CreateUser(ctx context.Context, id uuid.UUID, username, email, firstName, lastName, passwordHash string) (*sqlc.User, error) {
	fmt.Printf("CreateUser called: ID=%s, Username=%s, Email=%s\n", id.String(), username, email)

	user, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           id,
		Username:     username,
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: passwordHash,
	})
	if err != nil {
		fmt.Printf("CreateUser database error: %v\n", err)
		return nil, err
	}

	fmt.Printf("CreateUser successful: ID=%s, Username=%s\n", user.ID.String(), user.Username)
	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (r *PostgreSQLUserRepository) GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error) {
	user, err := r.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsernameWithPassword retrieves a user by username with password for authentication
func (r *PostgreSQLUserRepository) GetUserByUsernameWithPassword(ctx context.Context, username string) (*sqlc.User, error) {
	user, err := r.queries.GetUserByUsernameWithPassword(ctx, username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *PostgreSQLUserRepository) GetUserByEmail(ctx context.Context, email string) (*sqlc.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *PostgreSQLUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*sqlc.User, error) {
	user, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UserExistsByUsername checks if a user exists by username
func (r *PostgreSQLUserRepository) UserExistsByUsername(ctx context.Context, username string) (bool, error) {
	return r.queries.UserExistsByUsername(ctx, username)
}

// UserExistsByEmail checks if a user exists by email
func (r *PostgreSQLUserRepository) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.queries.UserExistsByEmail(ctx, email)
}

// GetAllUsersWithArtists retrieves all users with their artists for matching
func (r *PostgreSQLUserRepository) GetAllUsersWithArtists(ctx context.Context) ([]sqlc.GetUsersWithArtistsRow, error) {
	return r.queries.GetUsersWithArtists(ctx)
}

// CreateArtist creates or retrieves an artist by name
func (r *PostgreSQLUserRepository) CreateArtist(ctx context.Context, name string) (*sqlc.Artist, error) {
	artist, err := r.queries.CreateArtist(ctx, name)
	if err != nil {
		return nil, err
	}
	return &artist, nil
}

// GetArtistByName retrieves an artist by name
func (r *PostgreSQLUserRepository) GetArtistByName(ctx context.Context, name string) (*sqlc.Artist, error) {
	artist, err := r.queries.GetArtistByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &artist, nil
}

// SearchArtists performs fuzzy search for artists
func (r *PostgreSQLUserRepository) SearchArtists(ctx context.Context, query string, limit int32) ([]sqlc.Artist, error) {
	// Prepare fuzzy search patterns
	fuzzyPattern := "%" + strings.ToLower(query) + "%"
	exactQuery := strings.ToLower(query)
	startsWithPattern := strings.ToLower(query) + "%"

	return r.queries.SearchArtists(ctx, sqlc.SearchArtistsParams{
		Lower:   fuzzyPattern,      // LIKE pattern for general fuzzy matching
		Lower_2: exactQuery,        // Exact match for highest priority
		Lower_3: startsWithPattern, // Starts with for second priority
		Limit:   limit,             // Limit results
	})
}

// SetUserArtists sets the complete list of artists for a user
func (r *PostgreSQLUserRepository) SetUserArtists(ctx context.Context, userID uuid.UUID, artistNames []string) error {
	// First clear existing associations
	if err := r.queries.ClearUserArtists(ctx, userID); err != nil {
		return err
	}

	// Set new associations
	// Create a slice of integers from 1 to len(artistNames)
	orderValues := make([]int32, len(artistNames))
	for i := range orderValues {
		orderValues[i] = int32(i + 1)
	}

	return r.queries.SetUserArtists(ctx, sqlc.SetUserArtistsParams{
		UserID:  userID,
		Column2: artistNames,
		Column3: orderValues,
	})
}

// GetUserArtists retrieves all artists for a user
func (r *PostgreSQLUserRepository) GetUserArtists(ctx context.Context, userID uuid.UUID) ([]sqlc.Artist, error) {
	return r.queries.GetUserArtists(ctx, userID)
}

// ClearUserArtists removes all artist associations for a user
func (r *PostgreSQLUserRepository) ClearUserArtists(ctx context.Context, userID uuid.UUID) error {
	return r.queries.ClearUserArtists(ctx, userID)
}

// FindSimilarUsers finds users similar to the given username based on their artist preferences
func (repo *PostgreSQLUserRepository) FindSimilarUsers(ctx context.Context, username string) ([]sqlc.FindSimilarUsersRow, error) {
	return repo.queries.FindSimilarUsers(ctx, username)
}

// Feedback operations
func (repo *PostgreSQLUserRepository) CreateFeedback(ctx context.Context, userID uuid.UUID, feedbackText string) (*sqlc.Feedback, error) {
	params := sqlc.CreateFeedbackParams{
		UserID:       userID,
		FeedbackText: feedbackText,
	}
	feedback, err := repo.queries.CreateFeedback(ctx, params)
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (repo *PostgreSQLUserRepository) GetFeedbackByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Feedback, error) {
	return repo.queries.GetFeedbackByUser(ctx, userID)
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
