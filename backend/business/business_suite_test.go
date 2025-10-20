package business_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	business "github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBusiness(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Business Suite")
}

// MockEventProvider implements concert.EventProvider for testing
type MockEventProvider struct {
	events []concert.Event
	errors map[string]error
}

func (m *MockEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	if err, exists := m.errors["search"]; exists {
		return nil, err
	}

	return &concert.SearchResult{
		Events:      m.events,
		TotalCount:  len(m.events),
		CurrentPage: 0,
		TotalPages:  1,
		HasMore:     false,
	}, nil
}

func (m *MockEventProvider) GetEventByID(ctx context.Context, id string) (*concert.Event, error) {
	if err, exists := m.errors["getById"]; exists {
		return nil, err
	}

	for _, event := range m.events {
		if event.ID == id {
			return &event, nil
		}
	}

	return nil, nil
}

func (m *MockEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	if err, exists := m.errors["getByArtist"]; exists {
		return nil, err
	}

	var filteredEvents []concert.Event
	for _, event := range m.events {
		for _, artist := range event.Artists {
			if artist.Name == artistName {
				filteredEvents = append(filteredEvents, event)
				break
			}
		}
	}

	return &concert.SearchResult{
		Events:      filteredEvents,
		TotalCount:  len(filteredEvents),
		CurrentPage: 0,
		TotalPages:  1,
		HasMore:     false,
	}, nil
}

func (m *MockEventProvider) IsHealthy(ctx context.Context) error {
	if err, exists := m.errors["health"]; exists {
		return err
	}
	return nil
}

func (m *MockEventProvider) GetProviderName() string {
	return "MockProvider"
}

func (m *MockEventProvider) SetEvents(events []concert.Event) {
	m.events = events
}

func (m *MockEventProvider) SetError(operation string, err error) {
	if m.errors == nil {
		m.errors = make(map[string]error)
	}
	m.errors[operation] = err
}

func (m *MockEventProvider) AddEvent(event concert.Event) {
	m.events = append(m.events, event)
}

// MockUser represents a user for testing
type MockUser struct {
	ID             string
	Username       string
	Email          string
	FirstName      string
	LastName       string
	Program        string
	GraduationYear int32
}

// MockUserWithPassword represents a user with password for testing
type MockUserWithPassword struct {
	MockUser
	PasswordHash string
}

// MockUserRepository implements UserRepository for testing
type MockUserRepository struct {
	userExistsByUsername      map[string]bool
	userExistsByEmail         map[string]bool
	userExistsByUsernameError error
	userExistsByEmailError    error
	createUserResult          *MockUser
	createUserError           error
	getUserResult             *MockUserWithPassword
	getUserError              error
	getUserArtistsResult      []string
	getUserArtistsError       error
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		userExistsByUsername: make(map[string]bool),
		userExistsByEmail:    make(map[string]bool),
	}
}

func (m *MockUserRepository) SetUserExistsByUsername(username string, exists bool, err error) {
	m.userExistsByUsername[username] = exists
	m.userExistsByUsernameError = err
}

func (m *MockUserRepository) SetUserExistsByEmail(email string, exists bool, err error) {
	m.userExistsByEmail[email] = exists
	m.userExistsByEmailError = err
}

func (m *MockUserRepository) SetCreateUserResult(user *MockUser, err error) {
	m.createUserResult = user
	m.createUserError = err
}

func (m *MockUserRepository) SetGetUserByUsernameWithPasswordResult(user *MockUserWithPassword, err error) {
	m.getUserResult = user
	m.getUserError = err
}

func (m *MockUserRepository) SetGetUserArtistsResult(artists []string, err error) {
	m.getUserArtistsResult = artists
	m.getUserArtistsError = err
}

// UserRepository interface implementation
func (m *MockUserRepository) CreateUser(ctx context.Context, id uuid.UUID, username, email, firstName, lastName, passwordHash, program string, graduationYear int32) (*sqlc.User, error) {
	if m.createUserError != nil {
		return nil, m.createUserError
	}
	if m.createUserResult != nil {
		return &sqlc.User{
			ID:             id,
			Username:       username,
			Email:          email,
			FirstName:      firstName,
			LastName:       lastName,
			PasswordHash:   passwordHash,
			Program:        pgtype.Text{String: program, Valid: program != ""},
			GraduationYear: pgtype.Int4{Int32: graduationYear, Valid: graduationYear != 0},
		}, nil
	}
	return nil, nil
}

func (m *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error) {
	if m.getUserError != nil {
		return nil, m.getUserError
	}
	if m.getUserResult != nil && m.getUserResult.Username == username {
		return &sqlc.User{
			ID:        uuid.MustParse(m.getUserResult.ID),
			Username:  m.getUserResult.Username,
			Email:     m.getUserResult.Email,
			FirstName: m.getUserResult.FirstName,
			LastName:  m.getUserResult.LastName,
		}, nil
	}
	return nil, nil
}

func (m *MockUserRepository) GetUserByUsernameWithPassword(ctx context.Context, username string) (*sqlc.User, error) {
	if m.getUserError != nil {
		return nil, m.getUserError
	}
	if m.getUserResult != nil && m.getUserResult.Username == username {
		return &sqlc.User{
			ID:           uuid.MustParse(m.getUserResult.ID),
			Username:     m.getUserResult.Username,
			Email:        m.getUserResult.Email,
			FirstName:    m.getUserResult.FirstName,
			LastName:     m.getUserResult.LastName,
			PasswordHash: m.getUserResult.PasswordHash,
		}, nil
	}
	return nil, nil
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*sqlc.User, error) {
	return nil, nil // Not implemented for these tests
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*sqlc.User, error) {
	return nil, nil // Not implemented for these tests
}

// Web Push subscription methods to satisfy interface for auth integration tests
func (m *MockUserRepository) UpsertPushSubscription(ctx context.Context, userID *uuid.UUID, endpoint, p256dh, auth, userAgent string) error {
	return nil
}
func (m *MockUserRepository) GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error) {
	return []sqlc.PushSubscription{}, nil
}
func (m *MockUserRepository) GetAnyPushSubscriptions(ctx context.Context, lim int32) ([]sqlc.PushSubscription, error) {
	return []sqlc.PushSubscription{}, nil
}
func (m *MockUserRepository) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	return nil
}

// Satisfy interface: enumerate distinct push-enabled user IDs (unused in these tests)
func (m *MockUserRepository) GetDistinctPushUserIDs(ctx context.Context, limit, offset int32) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

func (m *MockUserRepository) UserExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.userExistsByUsernameError != nil {
		return false, m.userExistsByUsernameError
	}
	exists, ok := m.userExistsByUsername[username]
	return exists && ok, nil
}

func (m *MockUserRepository) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.userExistsByEmailError != nil {
		return false, m.userExistsByEmailError
	}
	exists, ok := m.userExistsByEmail[email]
	return exists && ok, nil
}

// FindSimilarUsers removed with legacy similarity system; no-op placeholder omitted.

func (m *MockUserRepository) SaveFeedback(ctx context.Context, userID uuid.UUID, feedback string) (*sqlc.Feedback, error) {
	return nil, nil // Not implemented for these tests
}

func (m *MockUserRepository) CreateFeedback(ctx context.Context, userID uuid.UUID, feedback string) (*sqlc.Feedback, error) {
	return nil, nil // Not implemented for these tests
}

func (m *MockUserRepository) GetFeedbackByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Feedback, error) {
	return nil, nil // Not implemented for these tests
}

// Password Reset methods - stub implementations for existing tests
func (m *MockUserRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (sqlc.PasswordResetToken, error) {
	return sqlc.PasswordResetToken{}, nil // Not implemented for these tests
}

func (m *MockUserRepository) GetPasswordResetToken(ctx context.Context, token string) (sqlc.PasswordResetToken, error) {
	return sqlc.PasswordResetToken{}, nil // Not implemented for these tests
}

func (m *MockUserRepository) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	return nil // Not implemented for these tests
}

func (m *MockUserRepository) MarkPasswordResetTokenAsUsed(ctx context.Context, token string) error {
	return nil // Not implemented for these tests
}

func (m *MockUserRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) (sqlc.UpdateUserPasswordRow, error) {
	return sqlc.UpdateUserPasswordRow{}, nil // Not implemented for these tests
}

func (m *MockUserRepository) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	return nil // Not implemented for these tests
}

func (m *MockUserRepository) DeleteUserPasswordResetTokens(ctx context.Context, userID uuid.UUID) error {
	return nil // Not implemented for these tests
}

// Spotify token operations (unused in these tests)
func (m *MockUserRepository) UpsertSpotifyTokens(ctx context.Context, userID uuid.UUID, accessToken string, refreshTokenEncrypted []byte, expiresAt time.Time, scope string, tokenType string) error {
	return nil
}
func (m *MockUserRepository) GetSpotifyTokensByUser(ctx context.Context, userID uuid.UUID) (*sqlc.SpotifyToken, error) {
	return nil, nil
}

// Snapshot storage no-ops for tests not covering Spotify yet
func (m *MockUserRepository) StoreSpotifyTopArtists(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []business.SpotifyTopArtist) error {
	return nil
}
func (m *MockUserRepository) StoreSpotifyTopTracks(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []business.SpotifyTopTrack) error {
	return nil
}

// FindSimilarUsersBySpotifyTopArtists mock returns empty slice (tests that need real data use real repo)
func (m *MockUserRepository) FindSimilarUsersBySpotifyTopArtists(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]business.SimilarUserResult, error) {
	return []business.SimilarUserResult{}, nil
}
func (m *MockUserRepository) FindSimilarUsersBySpotifyTopArtistsFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string) ([]business.SimilarUserResult, error) {
	return []business.SimilarUserResult{}, nil
}

// FindSimilarUsersBySpotifyTopTracks mock returns empty slice
func (m *MockUserRepository) FindSimilarUsersBySpotifyTopTracks(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]business.SimilarUserResult, error) {
	return []business.SimilarUserResult{}, nil
}
func (m *MockUserRepository) FindSimilarUsersBySpotifyTopTracksFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string) ([]business.SimilarUserResult, error) {
	return []business.SimilarUserResult{}, nil
}
