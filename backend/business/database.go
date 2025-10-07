package business

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

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

	// (Legacy manual artist operations removed in Spotify mode)

	// Matching operations
	// (Similar user matching removed with legacy artist system; placeholder for future Spotify-based matching)

	// Feedback operations
	CreateFeedback(ctx context.Context, userID uuid.UUID, feedbackText string) (*sqlc.Feedback, error)
	GetFeedbackByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Feedback, error)

	// Password reset operations
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (sqlc.PasswordResetToken, error)
	GetPasswordResetToken(ctx context.Context, token string) (sqlc.PasswordResetToken, error)
	MarkPasswordResetTokenAsUsed(ctx context.Context, token string) error
	DeleteExpiredPasswordResetTokens(ctx context.Context) error
	DeleteUserPasswordResetTokens(ctx context.Context, userID uuid.UUID) error
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) (sqlc.UpdateUserPasswordRow, error)

	// Spotify token operations
	UpsertSpotifyTokens(ctx context.Context, userID uuid.UUID, accessToken string, refreshTokenEncrypted []byte, expiresAt time.Time, scope string, tokenType string) error
	GetSpotifyTokensByUser(ctx context.Context, userID uuid.UUID) (*sqlc.SpotifyToken, error)
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
	tr := otel.Tracer("repo.user")
	ctx, span := tr.Start(ctx, "CreateUser")
	span.SetAttributes(
		attribute.String("db.system", "postgres"),
		attribute.String("db.operation", "CreateUser"),
		attribute.String("app.username", username),
	)
	defer span.End()
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		fmt.Printf("CreateUser database error: %v\n", err)
		return nil, err
	}
	fmt.Printf("CreateUser successful: ID=%s, Username=%s\n", user.ID.String(), user.Username)
	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (r *PostgreSQLUserRepository) GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error) {
	tr := otel.Tracer("repo.user")
	ctx, span := tr.Start(ctx, "GetUserByUsername")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.username", username))
	defer span.End()
	user, err := r.queries.GetUserByUsername(ctx, username)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return &user, nil
}

// GetUserByUsernameWithPassword retrieves a user by username with password for authentication
func (r *PostgreSQLUserRepository) GetUserByUsernameWithPassword(ctx context.Context, username string) (*sqlc.User, error) {
	tr := otel.Tracer("repo.user")
	ctx, span := tr.Start(ctx, "GetUserByUsernameWithPassword")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.username", username))
	defer span.End()
	user, err := r.queries.GetUserByUsernameWithPassword(ctx, username)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *PostgreSQLUserRepository) GetUserByEmail(ctx context.Context, email string) (*sqlc.User, error) {
	tr := otel.Tracer("repo.user")
	ctx, span := tr.Start(ctx, "GetUserByEmail")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.email", email))
	defer span.End()
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *PostgreSQLUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*sqlc.User, error) {
	tr := otel.Tracer("repo.user")
	ctx, span := tr.Start(ctx, "GetUserByID")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.user_id", id.String()))
	defer span.End()
	user, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return &user, nil
}

// UserExistsByUsername checks if a user exists by username
func (r *PostgreSQLUserRepository) UserExistsByUsername(ctx context.Context, username string) (bool, error) {
	tr := otel.Tracer("repo.user")
	ctx, span := tr.Start(ctx, "UserExistsByUsername")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.username", username))
	defer span.End()
	return r.queries.UserExistsByUsername(ctx, username)
}

// UserExistsByEmail checks if a user exists by email
func (r *PostgreSQLUserRepository) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	tr := otel.Tracer("repo.user")
	ctx, span := tr.Start(ctx, "UserExistsByEmail")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.email", email))
	defer span.End()
	return r.queries.UserExistsByEmail(ctx, email)
}

// (FindSimilarUsers removed; legacy manual artist similarity retired. Will be reintroduced using Spotify data.)

// Feedback operations
func (r *PostgreSQLUserRepository) CreateFeedback(ctx context.Context, userID uuid.UUID, feedbackText string) (*sqlc.Feedback, error) {
	tr := otel.Tracer("repo.feedback")
	ctx, span := tr.Start(ctx, "CreateFeedback")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.user_id", userID.String()))
	defer span.End()
	params := sqlc.CreateFeedbackParams{
		UserID:       userID,
		FeedbackText: feedbackText,
	}
	feedback, err := r.queries.CreateFeedback(ctx, params)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return &feedback, nil
}

func (r *PostgreSQLUserRepository) GetFeedbackByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Feedback, error) {
	tr := otel.Tracer("repo.feedback")
	ctx, span := tr.Start(ctx, "GetFeedbackByUser")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.user_id", userID.String()))
	defer span.End()
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

// CreatePasswordResetToken creates a new password reset token
func (r *PostgreSQLUserRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (sqlc.PasswordResetToken, error) {
	tr := otel.Tracer("repo.password")
	ctx, span := tr.Start(ctx, "CreatePasswordResetToken")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.user_id", userID.String()))
	defer span.End()
	return r.queries.CreatePasswordResetToken(ctx, sqlc.CreatePasswordResetTokenParams{
		UserID:    userID,
		Token:     token,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
}

// GetPasswordResetToken retrieves a password reset token if valid
func (r *PostgreSQLUserRepository) GetPasswordResetToken(ctx context.Context, token string) (sqlc.PasswordResetToken, error) {
	tr := otel.Tracer("repo.password")
	ctx, span := tr.Start(ctx, "GetPasswordResetToken")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.token", token))
	defer span.End()
	return r.queries.GetPasswordResetToken(ctx, token)
}

// MarkPasswordResetTokenAsUsed marks a token as used
func (r *PostgreSQLUserRepository) MarkPasswordResetTokenAsUsed(ctx context.Context, token string) error {
	tr := otel.Tracer("repo.password")
	ctx, span := tr.Start(ctx, "MarkPasswordResetTokenAsUsed")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.token", token))
	defer span.End()
	return r.queries.MarkPasswordResetTokenAsUsed(ctx, token)
}

// DeleteExpiredPasswordResetTokens removes expired tokens
func (r *PostgreSQLUserRepository) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	tr := otel.Tracer("repo.password")
	ctx, span := tr.Start(ctx, "DeleteExpiredPasswordResetTokens")
	span.SetAttributes(attribute.String("db.system", "postgres"))
	defer span.End()
	return r.queries.DeleteExpiredPasswordResetTokens(ctx)
}

// DeleteUserPasswordResetTokens removes all reset tokens for a user
func (r *PostgreSQLUserRepository) DeleteUserPasswordResetTokens(ctx context.Context, userID uuid.UUID) error {
	tr := otel.Tracer("repo.password")
	ctx, span := tr.Start(ctx, "DeleteUserPasswordResetTokens")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.user_id", userID.String()))
	defer span.End()
	return r.queries.DeleteUserPasswordResetTokens(ctx, userID)
}

// UpdateUserPassword updates a user's password
func (r *PostgreSQLUserRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) (sqlc.UpdateUserPasswordRow, error) {
	tr := otel.Tracer("repo.password")
	ctx, span := tr.Start(ctx, "UpdateUserPassword")
	span.SetAttributes(attribute.String("db.system", "postgres"), attribute.String("app.user_id", userID.String()))
	defer span.End()
	return r.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: passwordHash,
	})
}

// UpsertSpotifyTokens stores or updates a user's Spotify tokens (refresh token already encrypted by caller)
func (r *PostgreSQLUserRepository) UpsertSpotifyTokens(ctx context.Context, userID uuid.UUID, accessToken string, refreshTokenEncrypted []byte, expiresAt time.Time, scope string, tokenType string) error {
	params := sqlc.UpsertSpotifyTokensParams{
		UserID:                userID,
		AccessToken:           accessToken,
		RefreshTokenEncrypted: refreshTokenEncrypted,
		ExpiresAt:             pgtype.Timestamptz{Time: expiresAt, Valid: true},
		Scope:                 pgtype.Text{String: scope, Valid: scope != ""},
		TokenType:             tokenType,
	}
	return r.queries.UpsertSpotifyTokens(ctx, params)
}

// GetSpotifyTokensByUser retrieves stored (encrypted) Spotify tokens for user
func (r *PostgreSQLUserRepository) GetSpotifyTokensByUser(ctx context.Context, userID uuid.UUID) (*sqlc.SpotifyToken, error) {
	tok, err := r.queries.GetSpotifyTokensByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &tok, nil
}
