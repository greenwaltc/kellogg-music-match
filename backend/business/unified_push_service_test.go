package business

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	apns "github.com/sideshow/apns2"
)

// stubRepo implements UserRepository with no-ops and adds device token methods used via type assertions
type stubRepo struct {
	tokens  map[uuid.UUID][]DeviceToken
	deleted []struct {
		uid             uuid.UUID
		platform, token string
	}
}

// Device token narrow methods
func (s *stubRepo) ListDeviceTokensByUser(ctx context.Context, userID uuid.UUID) ([]DeviceToken, error) {
	return s.tokens[userID], nil
}
func (s *stubRepo) DeleteDeviceToken(ctx context.Context, userID uuid.UUID, platform, token string) error {
	s.deleted = append(s.deleted, struct {
		uid             uuid.UUID
		platform, token string
	}{userID, platform, token})
	return nil
}

// UserRepository methods (no-ops for tests)
func (s *stubRepo) CreateUser(ctx context.Context, id uuid.UUID, username, email, firstName, lastName, passwordHash, program string, graduationYear int32) (*sqlc.User, error) {
	return &sqlc.User{ID: id, Username: username, Email: email, FirstName: firstName, LastName: lastName, Program: pgtype.Text{}, GraduationYear: pgtype.Int4{}}, nil
}
func (s *stubRepo) GetUserByUsername(ctx context.Context, username string) (*sqlc.User, error) {
	return nil, nil
}
func (s *stubRepo) GetUserByUsernameWithPassword(ctx context.Context, username string) (*sqlc.User, error) {
	return nil, nil
}
func (s *stubRepo) GetUserByEmail(ctx context.Context, email string) (*sqlc.User, error) {
	return nil, nil
}
func (s *stubRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*sqlc.User, error) {
	return nil, nil
}
func (s *stubRepo) UserExistsByUsername(ctx context.Context, username string) (bool, error) {
	return false, nil
}
func (s *stubRepo) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}
func (s *stubRepo) CreateFeedback(ctx context.Context, userID uuid.UUID, feedbackText string) (*sqlc.Feedback, error) {
	return nil, nil
}
func (s *stubRepo) GetFeedbackByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.Feedback, error) {
	return nil, nil
}
func (s *stubRepo) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (sqlc.PasswordResetToken, error) {
	return sqlc.PasswordResetToken{}, nil
}
func (s *stubRepo) GetPasswordResetToken(ctx context.Context, token string) (sqlc.PasswordResetToken, error) {
	return sqlc.PasswordResetToken{}, nil
}
func (s *stubRepo) MarkPasswordResetTokenAsUsed(ctx context.Context, token string) error { return nil }
func (s *stubRepo) DeleteExpiredPasswordResetTokens(ctx context.Context) error           { return nil }
func (s *stubRepo) DeleteUserPasswordResetTokens(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (s *stubRepo) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) (sqlc.UpdateUserPasswordRow, error) {
	return sqlc.UpdateUserPasswordRow{ID: userID}, nil
}
func (s *stubRepo) UpsertSpotifyTokens(ctx context.Context, userID uuid.UUID, accessToken string, refreshTokenEncrypted []byte, expiresAt time.Time, scope string, tokenType string) error {
	return nil
}
func (s *stubRepo) GetSpotifyTokensByUser(ctx context.Context, userID uuid.UUID) (*sqlc.SpotifyToken, error) {
	return nil, nil
}
func (s *stubRepo) StoreSpotifyTopArtists(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []SpotifyTopArtist) error {
	return nil
}
func (s *stubRepo) StoreSpotifyTopTracks(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []SpotifyTopTrack) error {
	return nil
}
func (s *stubRepo) GetUserTopArtistsByRange(ctx context.Context, userID uuid.UUID, rng string, limit, offset int32) ([]SpotifyTopArtist, error) {
	return []SpotifyTopArtist{}, nil
}
func (s *stubRepo) GetUserTopTracksByRange(ctx context.Context, userID uuid.UUID, rng string, limit, offset int32) ([]SpotifyTopTrack, error) {
	return []SpotifyTopTrack{}, nil
}
func (s *stubRepo) FindSimilarUsersBySpotifyTopArtists(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, includeDetails bool) ([]SimilarUserResult, error) {
	return nil, nil
}
func (s *stubRepo) FindSimilarUsersBySpotifyTopArtistsFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string, includeDetails bool) ([]SimilarUserResult, error) {
	return nil, nil
}
func (s *stubRepo) FindSimilarUsersBySpotifyTopTracks(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, includeDetails bool) ([]SimilarUserResult, error) {
	return nil, nil
}
func (s *stubRepo) FindSimilarUsersBySpotifyTopTracksFiltered(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32, nameFilter string, includeDetails bool) ([]SimilarUserResult, error) {
	return nil, nil
}
func (s *stubRepo) UpsertPushSubscription(ctx context.Context, userID *uuid.UUID, endpoint, p256dh, auth, userAgent string) error {
	return nil
}
func (s *stubRepo) GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error) {
	return nil, nil
}
func (s *stubRepo) GetAnyPushSubscriptions(ctx context.Context, lim int32) ([]sqlc.PushSubscription, error) {
	return nil, nil
}
func (s *stubRepo) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	return nil
}
func (s *stubRepo) GetDistinctPushUserIDs(ctx context.Context, limit, offset int32) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

func TestAPNsPrunesInvalidToken(t *testing.T) {
	cfg := &config.Config{NativePush: config.NativePushConfig{Enabled: true, APNs: config.APNsConfig{Enabled: true, BundleID: "com.example.app"}}}
	repo := &stubRepo{tokens: map[uuid.UUID][]DeviceToken{}}
	svc := NewUnifiedPushService(cfg, repo)
	// Inject a fake apns push returning BadDeviceToken
	svc.apnsPushFunc = func(req *apns.Notification) (*apns.Response, error) {
		return &apns.Response{StatusCode: 400, Reason: "BadDeviceToken"}, nil
	}
	userID := uuid.New()
	repo.tokens[userID] = []DeviceToken{{UserID: userID, Platform: "ios", Token: "dead"}}
	n := WebPushNotification{Title: "t", Body: "b"}
	_ = svc.SendToUser(context.Background(), userID, n)
	if len(repo.deleted) != 1 || repo.deleted[0].token != "dead" || repo.deleted[0].platform != "ios" {
		t.Fatalf("expected one deleted ios token, got %+v", repo.deleted)
	}
}

func TestFCMPrunesOn404(t *testing.T) {
	cfg := &config.Config{NativePush: config.NativePushConfig{Enabled: true, FCM: config.FCMConfig{Enabled: true, ProjectID: "demo"}}}
	repo := &stubRepo{tokens: map[uuid.UUID][]DeviceToken{}}
	svc := NewUnifiedPushService(cfg, repo)
	// Inject a fake http Do
	svc.fcmDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Body: ioNopCloser{Reader: strings.NewReader("")}}, nil
	}
	userID := uuid.New()
	repo.tokens[userID] = []DeviceToken{{UserID: userID, Platform: "android", Token: "gone"}}
	n := WebPushNotification{Title: "t", Body: "b"}
	_ = svc.SendToUser(context.Background(), userID, n)
	if len(repo.deleted) != 1 || repo.deleted[0].token != "gone" || repo.deleted[0].platform != "android" {
		t.Fatalf("expected one deleted android token, got %+v", repo.deleted)
	}
}

func TestFCM400NonUnregisteredDoesNotPrune(t *testing.T) {
	cfg := &config.Config{NativePush: config.NativePushConfig{Enabled: true, FCM: config.FCMConfig{Enabled: true, ProjectID: "demo"}}}
	repo := &stubRepo{tokens: map[uuid.UUID][]DeviceToken{}}
	svc := NewUnifiedPushService(cfg, repo)
	// Simulate a 400 with a non-UNREGISTERED error status
	body := `{"error":{"code":400,"status":"INVALID_ARGUMENT","message":"bad args","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"SENDER_ID_MISMATCH"}]}}`
	svc.fcmDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 400, Body: ioNopCloser{Reader: strings.NewReader(body)}}, nil
	}
	userID := uuid.New()
	repo.tokens[userID] = []DeviceToken{{UserID: userID, Platform: "android", Token: "maybe-still-valid"}}
	n := WebPushNotification{Title: "t", Body: "b"}
	_ = svc.SendToUser(context.Background(), userID, n)
	if len(repo.deleted) != 0 {
		t.Fatalf("expected no deletions for non-UNREGISTERED 400, got %+v", repo.deleted)
	}
}

// simple ReadCloser wrapper for http.Response bodies
type ioNopCloser struct{ Reader *strings.Reader }

func (c ioNopCloser) Read(p []byte) (int, error) { return c.Reader.Read(p) }
func (c ioNopCloser) Close() error               { return nil }
