package business

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// User operations
	CreateUser(ctx context.Context, id uuid.UUID, username, email, firstName, lastName, passwordHash, program string, graduationYear int32) (*sqlc.User, error)
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
	GetUserArtists(ctx context.Context, userID uuid.UUID) ([]sqlc.GetUserArtistsRow, error)
	ClearUserArtists(ctx context.Context, userID uuid.UUID) error

	// Matching operations
	FindSimilarUsers(ctx context.Context, username string) ([]sqlc.FindSimilarUsersRow, error)

	// Feedback operations
	CreateFeedback(ctx context.Context, userID uuid.UUID, feedbackText string) (*sqlc.Feedback, error)
	GetFeedbackByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Feedback, error)
}

// PostgreSQLUserRepository implements UserRepository using pgxpool
type PostgreSQLUserRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewPostgreSQLUserRepository creates a new repository backed by pgx/v5
func NewPostgreSQLUserRepository(cfg *config.DatabaseConfig) (*PostgreSQLUserRepository, error) {
	dsn := cfg.ConnectionString()

	// Using default pgxpool config; customize if you want timeouts/limits
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgx ping: %w", err)
	}

	// Enable testing-mode guard in the database session when running tests to avoid deadlocks
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" || strings.Contains(strings.ToLower(os.Args[0]), ".test") {
		// Best-effort: set a custom GUC that our trigger reads
		_, _ = pool.Exec(context.Background(), "SET kmm.testing_mode = 'on'")
	}

	return &PostgreSQLUserRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}, nil
}

// NewUserRepository creates a new UserRepository with default database configuration
func NewUserRepository() (UserRepository, error) {
	cfg := config.Load()
	return NewUserRepositoryWithConfig(&cfg.Database)
}

// NewUserRepositoryWithConfig creates a new UserRepository with provided database configuration
func NewUserRepositoryWithConfig(dbConfig *config.DatabaseConfig) (UserRepository, error) {
	// Log connection attempt (without password)
	fmt.Printf("Attempting database connection to %s:%s@%s:%s/%s\n",
		dbConfig.User, "***", dbConfig.Host, dbConfig.Port, dbConfig.Name)

	repo, err := NewPostgreSQLUserRepository(dbConfig)
	if err != nil {
		fmt.Printf("Database connection failed: %v\n", err)
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}

	fmt.Println("Database connection successful!")
	return repo, nil
}

// Close closes the database pool
func (r *PostgreSQLUserRepository) Close() error {
	if r.pool != nil {
		r.pool.Close()
	}
	return nil
}

// CreateUser creates a new user in the database
func (r *PostgreSQLUserRepository) CreateUser(ctx context.Context, id uuid.UUID, username, email, firstName, lastName, passwordHash, program string, graduationYear int32) (*sqlc.User, error) {
	fmt.Printf("CreateUser called: ID=%s, Username=%s, Email=%s, Program=%s, GradYear=%d\n",
		id.String(), username, email, program, graduationYear)

	params := sqlc.CreateUserParams{
		ID:           id,
		Username:     username,
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: passwordHash,

		// For pgx/v5, sqlc usually emits pgtype.* for nullable columns
		Program: pgtype.Text{
			String: program,
			Valid:  program != "",
		},
		GraduationYear: pgtype.Int4{
			Int32: graduationYear,
			Valid: graduationYear > 0,
		},
	}

	user, err := r.queries.CreateUser(ctx, params)
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
		SearchTerm:   fuzzyPattern,      // LIKE pattern for general fuzzy matching
		ExactMatch:   exactQuery,        // Exact match for highest priority
		PartialMatch: startsWithPattern, // Starts with for second priority
		Lim:          limit,             // Limit results
	})
}

// SetUserArtists sets the complete list of artists for a user
func (r *PostgreSQLUserRepository) SetUserArtists(ctx context.Context, userID uuid.UUID, artistNames []string) error {
	// First clear existing associations
	if err := r.queries.ClearUserArtists(ctx, userID); err != nil {
		return err
	}

	// Set new associations
	orderValues := make([]int32, len(artistNames))
	for i := range orderValues {
		orderValues[i] = int32(i + 1)
	}

	// sqlc generated SetUserArtistsParams currently exposes fields: UserID, Column2 (artist names), Column3 (ranks).
	return r.queries.SetUserArtists(ctx, sqlc.SetUserArtistsParams{
		UserID:      userID,
		ArtistNames: artistNames,
		Ranks:       orderValues,
	})
}

// GetUserArtists retrieves all artists for a user
func (r *PostgreSQLUserRepository) GetUserArtists(ctx context.Context, userID uuid.UUID) ([]sqlc.GetUserArtistsRow, error) {
	return r.queries.GetUserArtists(ctx, userID)
}

// ClearUserArtists removes all artist associations for a user
func (r *PostgreSQLUserRepository) ClearUserArtists(ctx context.Context, userID uuid.UUID) error {
	return r.queries.ClearUserArtists(ctx, userID)
}

// FindSimilarUsers finds users similar to the given username based on their artist preferences
func (r *PostgreSQLUserRepository) FindSimilarUsers(ctx context.Context, username string) ([]sqlc.FindSimilarUsersRow, error) {
	// Using the Chamfer-based finder we created (ensure your queries.sql has it)
	return r.queries.FindSimilarUsers(ctx, sqlc.FindSimilarUsersParams{
		Username: username,
		Lim:      20,
		TopK:     40,
		Alpha:    0.85,
	})
}

// Feedback operations
func (r *PostgreSQLUserRepository) CreateFeedback(ctx context.Context, userID uuid.UUID, feedbackText string) (*sqlc.Feedback, error) {
	params := sqlc.CreateFeedbackParams{
		UserID:       userID,
		FeedbackText: feedbackText,
	}
	feedback, err := r.queries.CreateFeedback(ctx, params)
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (r *PostgreSQLUserRepository) GetFeedbackByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Feedback, error) {
	return r.queries.GetFeedbackByUser(ctx, userID)
}

// Optional helper if you want to build the pool from env directly elsewhere
func NewPoolFromEnv() (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
