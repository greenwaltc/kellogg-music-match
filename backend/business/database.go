package business

import (
	"context"
	"encoding/json"
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

	"github.com/jackc/pgx/v5"
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

	// Spotify preference snapshot operations
	StoreSpotifyTopArtists(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []SpotifyTopArtist) error
	StoreSpotifyTopTracks(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []SpotifyTopTrack) error

	// Spotify similarity operations
	FindSimilarUsersBySpotifyTopArtists(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]SimilarUserResult, error)
	FindSimilarUsersBySpotifyTopArtistsFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string) ([]SimilarUserResult, error)
	FindSimilarUsersBySpotifyTopTracks(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]SimilarUserResult, error)
	FindSimilarUsersBySpotifyTopTracksFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string) ([]SimilarUserResult, error)

	// Web Push subscription operations
	UpsertPushSubscription(ctx context.Context, userID *uuid.UUID, endpoint, p256dh, auth, userAgent string) error
	GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error)
	GetAnyPushSubscriptions(ctx context.Context, lim int32) ([]sqlc.PushSubscription, error)
	DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
	// GetDistinctPushUserIDs returns distinct user IDs that have at least one stored push subscription.
	GetDistinctPushUserIDs(ctx context.Context, limit, offset int32) ([]uuid.UUID, error)
}

// PostgreSQLUserRepository implements UserRepository using pgxpool
type PostgreSQLUserRepository struct {
	pool                      *pgxpool.Pool
	queries                   *sqlc.Queries
	spotifyArtistsUpdatedHook func(uuid.UUID)
}

// DeviceToken is a lean business-layer struct
type DeviceToken struct {
	UserID      uuid.UUID
	Platform    string
	Token       string
	BundleID    *string
	AppPackage  *string
	DeviceModel *string
	OSVersion   *string
	AppVersion  *string
	LastSeenAt  time.Time
}

// Pool returns underlying pgx pool (primarily for integration tests)
func (r *PostgreSQLUserRepository) Pool() *pgxpool.Pool { return r.pool }

// Domain snapshot structs (lean, kept in business layer)
type SpotifyTopArtist struct {
	Rank            int32
	SpotifyArtistID string
	Name            string
	Genres          []string
	Popularity      *int32
	ImageURL        *string
}

type SpotifyTopTrack struct {
	Rank           int32
	SpotifyTrackID string
	Name           string
	ArtistNames    []string
	ArtistIDs      []string
	AlbumName      *string
	AlbumID        *string
	Popularity     *int32
	PreviewURL     *string
	DurationMS     *int32
	ImageURL       *string
}

// Overlap detail for a shared Spotify top artist between two users
type SpotifyArtistOverlap struct {
	SpotifyArtistID string `json:"spotify_artist_id"`
	Name            string `json:"name"`
	AnchorRank      int32  `json:"anchor_rank"`
	OtherRank       int32  `json:"other_rank"`
}

// SimilarUserResult represents a user similar to the anchor based on Spotify top artists
type SimilarUserResult struct {
	UserID         uuid.UUID
	Username       string
	FirstName      string
	LastName       string
	Program        *string
	GraduationYear *int32
	Similarity     float64
	Overlaps       []SpotifyArtistOverlap
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

// Web Push subscription operations
func (r *PostgreSQLUserRepository) UpsertPushSubscription(ctx context.Context, userID *uuid.UUID, endpoint, p256dh, auth, userAgent string) error {
	uid := pgtype.UUID{Valid: false}
	if userID != nil {
		uid = pgtype.UUID{Bytes: [16]byte{}, Valid: true}
		copy(uid.Bytes[:], userID[:])
	}
	return r.queries.UpsertPushSubscription(ctx, sqlc.UpsertPushSubscriptionParams{
		UserID:    uid,
		Endpoint:  endpoint,
		P256dh:    p256dh,
		Auth:      auth,
		UserAgent: pgtype.Text{String: userAgent, Valid: userAgent != ""},
	})
}

func (r *PostgreSQLUserRepository) GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error) {
	uid := pgtype.UUID{Bytes: [16]byte{}, Valid: true}
	copy(uid.Bytes[:], userID[:])
	return r.queries.GetPushSubscriptionsByUser(ctx, uid)
}

func (r *PostgreSQLUserRepository) GetAnyPushSubscriptions(ctx context.Context, lim int32) ([]sqlc.PushSubscription, error) {
	return r.queries.GetAnyPushSubscriptions(ctx, lim)
}

func (r *PostgreSQLUserRepository) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	return r.queries.DeletePushSubscriptionByEndpoint(ctx, endpoint)
}

// GetDistinctPushUserIDs enumerates distinct non-null user IDs that have at least one push subscription.
// Paged with limit/offset; order by user_id for stable pagination.
func (r *PostgreSQLUserRepository) GetDistinctPushUserIDs(ctx context.Context, limit, offset int32) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT user_id FROM push_subscriptions WHERE user_id IS NOT NULL ORDER BY user_id OFFSET $1 LIMIT $2`, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Native push device token operations (inline SQL for speed; can be moved to sqlc later)
func (r *PostgreSQLUserRepository) UpsertDeviceToken(ctx context.Context, userID uuid.UUID, platform, token, bundleID, appPackage, deviceModel, osVersion, appVersion string) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO push_device_tokens (user_id, platform, token, bundle_id, app_package, device_model, os_version, app_version, last_seen_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8, NOW())
ON CONFLICT (user_id, platform, token) DO UPDATE SET
  bundle_id = EXCLUDED.bundle_id,
  app_package = EXCLUDED.app_package,
  device_model = EXCLUDED.device_model,
  os_version = EXCLUDED.os_version,
  app_version = EXCLUDED.app_version,
  last_seen_at = NOW(),
  updated_at = NOW()`, userID, platform, token, nullIfEmpty(bundleID), nullIfEmpty(appPackage), nullIfEmpty(deviceModel), nullIfEmpty(osVersion), nullIfEmpty(appVersion))
	return err
}

func (r *PostgreSQLUserRepository) ListDeviceTokensByUser(ctx context.Context, userID uuid.UUID) ([]DeviceToken, error) {
	rows, err := r.pool.Query(ctx, `SELECT user_id, platform, token, bundle_id, app_package, device_model, os_version, app_version, COALESCE(last_seen_at, created_at) FROM push_device_tokens WHERE user_id=$1 ORDER BY last_seen_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeviceToken
	for rows.Next() {
		var dt DeviceToken
		var bundleID, appPkg, devModel, osVer, appVer *string
		if err := rows.Scan(&dt.UserID, &dt.Platform, &dt.Token, &bundleID, &appPkg, &devModel, &osVer, &appVer, &dt.LastSeenAt); err != nil {
			return nil, err
		}
		dt.BundleID = bundleID
		dt.AppPackage = appPkg
		dt.DeviceModel = devModel
		dt.OSVersion = osVer
		dt.AppVersion = appVer
		out = append(out, dt)
	}
	return out, rows.Err()
}

func (r *PostgreSQLUserRepository) DeleteDeviceToken(ctx context.Context, userID uuid.UUID, platform, token string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM push_device_tokens WHERE user_id=$1 AND platform=$2 AND token=$3`, userID, platform, token)
	return err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
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
		GraduationYear: pgtype.Int4{Int32: graduationYear, Valid: graduationYear > 0},
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

// StoreSpotifyTopArtists inserts/updates a snapshot for the given range & timestamp.
// Uses the same ON CONFLICT pattern as the sqlc query definition (duplicated here to avoid regeneration dependency during dev).
func (r *PostgreSQLUserRepository) StoreSpotifyTopArtists(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []SpotifyTopArtist) error {
	if rng == "" {
		rng = "medium_term"
	}
	batch := &pgx.Batch{}
	// We keep statement text small; ON CONFLICT ensures idempotency per rank.
	for _, it := range items {
		var popularity interface{} = nil
		if it.Popularity != nil {
			popularity = *it.Popularity
		}
		var img interface{} = nil
		if it.ImageURL != nil {
			img = *it.ImageURL
		}
		batch.Queue(`INSERT INTO spotify_top_artist_snapshots (user_id, fetched_at, range, item_rank, spotify_artist_id, name, genres, popularity, image_url)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (user_id, range, item_rank) DO UPDATE SET spotify_artist_id=EXCLUDED.spotify_artist_id, name=EXCLUDED.name, genres=EXCLUDED.genres, popularity=EXCLUDED.popularity, image_url=EXCLUDED.image_url, fetched_at=EXCLUDED.fetched_at`,
			userID, fetchedAt, rng, it.Rank, it.SpotifyArtistID, it.Name, it.Genres, popularity, img)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range items {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	// trigger cache invalidation hook if present
	if r.spotifyArtistsUpdatedHook != nil {
		r.spotifyArtistsUpdatedHook(userID)
	}
	return nil
}

// SetSpotifyArtistsUpdatedHook registers a callback fired after StoreSpotifyTopArtists successfully upserts data
func (r *PostgreSQLUserRepository) SetSpotifyArtistsUpdatedHook(fn func(uuid.UUID)) {
	r.spotifyArtistsUpdatedHook = fn
}

func (r *PostgreSQLUserRepository) StoreSpotifyTopTracks(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []SpotifyTopTrack) error {
	if rng == "" {
		rng = "medium_term"
	}
	batch := &pgx.Batch{}
	for _, it := range items {
		var albumName, albumID, preview, img interface{}
		var popularity, duration interface{}
		if it.AlbumName != nil {
			albumName = *it.AlbumName
		}
		if it.AlbumID != nil {
			albumID = *it.AlbumID
		}
		if it.PreviewURL != nil {
			preview = *it.PreviewURL
		}
		if it.ImageURL != nil {
			img = *it.ImageURL
		}
		if it.Popularity != nil {
			popularity = *it.Popularity
		}
		if it.DurationMS != nil {
			duration = *it.DurationMS
		}
		batch.Queue(`INSERT INTO spotify_top_track_snapshots (user_id, fetched_at, range, item_rank, spotify_track_id, name, artist_names, artist_ids, album_name, album_id, popularity, preview_url, duration_ms, image_url)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
ON CONFLICT (user_id, range, item_rank) DO UPDATE SET spotify_track_id=EXCLUDED.spotify_track_id, name=EXCLUDED.name, artist_names=EXCLUDED.artist_names, artist_ids=EXCLUDED.artist_ids, album_name=EXCLUDED.album_name, album_id=EXCLUDED.album_id, popularity=EXCLUDED.popularity, preview_url=EXCLUDED.preview_url, duration_ms=EXCLUDED.duration_ms, image_url=EXCLUDED.image_url, fetched_at=EXCLUDED.fetched_at`,
			userID, fetchedAt, rng, it.Rank, it.SpotifyTrackID, it.Name, it.ArtistNames, it.ArtistIDs, albumName, albumID, popularity, preview, duration, img)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range items {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// FindSimilarUsersBySpotifyTopArtists delegates to the sqlc-generated query and converts the JSON overlaps
func (r *PostgreSQLUserRepository) FindSimilarUsersBySpotifyTopArtists(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]SimilarUserResult, error) {
	if rng == "" {
		rng = "medium_term"
	}
	if limit <= 0 {
		limit = 10
	}
	// Load matching config to obtain per-user top N cap for ranks
	cfg := config.Load()
	rows, err := r.queries.FindTopNSimilarUsersBySpotifyArtists(ctx, sqlc.FindTopNSimilarUsersBySpotifyArtistsParams{
		LimitN:       limit,
		AnchorUserID: anchorUserID,
		Range:        rng,
		TopN:         int32(cfg.Matching.ArtistTopN),
	})
	if err != nil {
		return nil, err
	}
	results := make([]SimilarUserResult, 0, len(rows))
	for _, row := range rows {
		var overlaps []SpotifyArtistOverlap
		if len(row.OverlapsJson) > 0 {
			if err := json.Unmarshal(row.OverlapsJson, &overlaps); err != nil {
				// If decoding fails, continue but leave overlaps empty; log for debugging.
				fmt.Printf("WARN: failed to unmarshal overlaps_json for user %s: %v\n", row.UserID, err)
			}
		}
		var program *string
		if row.Program.Valid {
			program = &row.Program.String
		}
		var gradYear *int32
		if row.GraduationYear.Valid {
			gradYear = &row.GraduationYear.Int32
		}
		results = append(results, SimilarUserResult{
			UserID:         row.UserID,
			Username:       row.Username,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			Program:        program,
			GraduationYear: gradYear,
			Similarity:     row.Similarity,
			Overlaps:       overlaps,
		})
	}
	return results, nil
}

// FindSimilarUsersBySpotifyTopArtistsFiltered adds a fuzzy name filter over other users.
func (r *PostgreSQLUserRepository) FindSimilarUsersBySpotifyTopArtistsFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string) ([]SimilarUserResult, error) {
	if rng == "" {
		rng = "medium_term"
	}
	if limit <= 0 {
		limit = 10
	}
	cfg := config.Load()
	rows, err := r.queries.FindTopNSimilarUsersBySpotifyArtistsFiltered(ctx, sqlc.FindTopNSimilarUsersBySpotifyArtistsFilteredParams{
		LimitN:       limit,
		AnchorUserID: anchorUserID,
		Range:        rng,
		TopN:         int32(cfg.Matching.ArtistTopN),
		NameFilter:   pgtype.Text{String: nameFilter, Valid: nameFilter != ""},
	})
	if err != nil {
		return nil, err
	}
	results := make([]SimilarUserResult, 0, len(rows))
	for _, row := range rows {
		var overlaps []SpotifyArtistOverlap
		if len(row.OverlapsJson) > 0 {
			if err := json.Unmarshal(row.OverlapsJson, &overlaps); err != nil {
				fmt.Printf("WARN: failed to unmarshal overlaps_json for user %s: %v\n", row.UserID, err)
			}
		}
		var program *string
		if row.Program.Valid {
			program = &row.Program.String
		}
		var gradYear *int32
		if row.GraduationYear.Valid {
			gradYear = &row.GraduationYear.Int32
		}
		results = append(results, SimilarUserResult{
			UserID:         row.UserID,
			Username:       row.Username,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			Program:        program,
			GraduationYear: gradYear,
			Similarity:     row.Similarity,
			Overlaps:       overlaps,
		})
	}
	return results, nil
}

// FindSimilarUsersBySpotifyTopTracks delegates to the sqlc-generated track similarity query (feature-flag gated at higher layer)
func (r *PostgreSQLUserRepository) FindSimilarUsersBySpotifyTopTracks(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]SimilarUserResult, error) {
	if rng == "" {
		rng = "medium_term"
	}
	if limit <= 0 {
		limit = 10
	}
	cfg := config.Load()
	rows, err := r.queries.FindTopNSimilarUsersBySpotifyTracks(ctx, sqlc.FindTopNSimilarUsersBySpotifyTracksParams{
		LimitN:       limit,
		AnchorUserID: anchorUserID,
		Range:        rng,
		TopN:         int32(cfg.Matching.TrackTopN),
	})
	if err != nil {
		return nil, err
	}
	results := make([]SimilarUserResult, 0, len(rows))
	for _, row := range rows {
		// Reuse SpotifyArtistOverlap struct for now; semantic rename later if differentiating
		var overlaps []SpotifyArtistOverlap
		if len(row.OverlapsJson) > 0 {
			if err := json.Unmarshal(row.OverlapsJson, &overlaps); err != nil {
				fmt.Printf("WARN: failed to unmarshal track overlaps_json for user %s: %v\n", row.UserID, err)
			}
		}
		var program *string
		if row.Program.Valid {
			program = &row.Program.String
		}
		var gradYear *int32
		if row.GraduationYear.Valid {
			gradYear = &row.GraduationYear.Int32
		}
		results = append(results, SimilarUserResult{
			UserID:         row.UserID,
			Username:       row.Username,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			Program:        program,
			GraduationYear: gradYear,
			Similarity:     row.Similarity,
			Overlaps:       overlaps,
		})
	}
	return results, nil
}

// FindSimilarUsersBySpotifyTopTracksFiltered adds a fuzzy name filter over other users.
func (r *PostgreSQLUserRepository) FindSimilarUsersBySpotifyTopTracksFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string) ([]SimilarUserResult, error) {
	if rng == "" {
		rng = "medium_term"
	}
	if limit <= 0 {
		limit = 10
	}
	cfg := config.Load()
	rows, err := r.queries.FindTopNSimilarUsersBySpotifyTracksFiltered(ctx, sqlc.FindTopNSimilarUsersBySpotifyTracksFilteredParams{
		LimitN:       limit,
		AnchorUserID: anchorUserID,
		Range:        rng,
		TopN:         int32(cfg.Matching.TrackTopN),
		NameFilter:   pgtype.Text{String: nameFilter, Valid: nameFilter != ""},
	})
	if err != nil {
		return nil, err
	}
	results := make([]SimilarUserResult, 0, len(rows))
	for _, row := range rows {
		var overlaps []SpotifyArtistOverlap
		if len(row.OverlapsJson) > 0 {
			if err := json.Unmarshal(row.OverlapsJson, &overlaps); err != nil {
				fmt.Printf("WARN: failed to unmarshal track overlaps_json for user %s: %v\n", row.UserID, err)
			}
		}
		var program *string
		if row.Program.Valid {
			program = &row.Program.String
		}
		var gradYear *int32
		if row.GraduationYear.Valid {
			gradYear = &row.GraduationYear.Int32
		}
		results = append(results, SimilarUserResult{
			UserID:         row.UserID,
			Username:       row.Username,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			Program:        program,
			GraduationYear: gradYear,
			Similarity:     row.Similarity,
			Overlaps:       overlaps,
		})
	}
	return results, nil
}
