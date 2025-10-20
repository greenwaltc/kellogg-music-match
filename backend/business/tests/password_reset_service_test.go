package business_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Enhanced MockUserRepository for password reset testing
type EnhancedMockUserRepository struct {
	*MockUserRepository
	users               map[string]*sqlc.User
	passwordResetTokens map[string]sqlc.PasswordResetToken
	updatePasswordCalls []UpdatePasswordCall
}

type UpdatePasswordCall struct {
	UserID       uuid.UUID
	PasswordHash string
}

func NewEnhancedMockUserRepository() *EnhancedMockUserRepository {
	return &EnhancedMockUserRepository{MockUserRepository: NewMockUserRepository(), users: make(map[string]*sqlc.User), passwordResetTokens: make(map[string]sqlc.PasswordResetToken), updatePasswordCalls: make([]UpdatePasswordCall, 0)}
}

func (m *EnhancedMockUserRepository) SetUser(email string, user *sqlc.User) { m.users[email] = user }
func (m *EnhancedMockUserRepository) GetUserByEmail(ctx context.Context, email string) (*sqlc.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found")
}
func (m *EnhancedMockUserRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (sqlc.PasswordResetToken, error) {
	reset := sqlc.PasswordResetToken{ID: uuid.New(), UserID: userID, Token: token, ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true}, Used: pgtype.Bool{Bool: false, Valid: true}, CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}
	m.passwordResetTokens[token] = reset
	return reset, nil
}
func (m *EnhancedMockUserRepository) GetPasswordResetToken(ctx context.Context, token string) (sqlc.PasswordResetToken, error) {
	if rt, ok := m.passwordResetTokens[token]; ok {
		if rt.ExpiresAt.Time.After(time.Now()) && !rt.Used.Bool {
			return rt, nil
		}
	}
	return sqlc.PasswordResetToken{}, pgx.ErrNoRows
}
func (m *EnhancedMockUserRepository) GetPasswordResetTokenRaw(ctx context.Context, token string) (sqlc.PasswordResetToken, error) {
	if rt, ok := m.passwordResetTokens[token]; ok {
		return rt, nil
	}
	return sqlc.PasswordResetToken{}, pgx.ErrNoRows
}
func (m *EnhancedMockUserRepository) MarkPasswordResetTokenAsUsed(ctx context.Context, token string) error {
	if rt, ok := m.passwordResetTokens[token]; ok {
		rt.Used = pgtype.Bool{Bool: true, Valid: true}
		m.passwordResetTokens[token] = rt
		return nil
	}
	return fmt.Errorf("token not found")
}
func (m *EnhancedMockUserRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) (sqlc.UpdateUserPasswordRow, error) {
	m.updatePasswordCalls = append(m.updatePasswordCalls, UpdatePasswordCall{UserID: userID, PasswordHash: passwordHash})
	return sqlc.UpdateUserPasswordRow{ID: userID, Username: "testuser", Email: "test@example.com"}, nil
}
func (m *EnhancedMockUserRepository) DeleteUserPasswordResetTokens(ctx context.Context, userID uuid.UUID) error {
	for t, rt := range m.passwordResetTokens {
		if rt.UserID == userID && !rt.Used.Bool {
			delete(m.passwordResetTokens, t)
		}
	}
	return nil
}
func (m *EnhancedMockUserRepository) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	for t, rt := range m.passwordResetTokens {
		if rt.ExpiresAt.Time.Before(time.Now()) {
			delete(m.passwordResetTokens, t)
		}
	}
	return nil
}
// Satisfy interface method introduced for push notifications (unused here)
func (m *EnhancedMockUserRepository) GetDistinctPushUserIDs(ctx context.Context, limit, offset int32) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}
func (m *EnhancedMockUserRepository) UpsertSpotifyTokens(ctx context.Context, userID uuid.UUID, accessToken string, refreshTokenEncrypted []byte, expiresAt time.Time, scope string, tokenType string) error {
	return nil
}
func (m *EnhancedMockUserRepository) GetSpotifyTokensByUser(ctx context.Context, userID uuid.UUID) (*sqlc.SpotifyToken, error) {
	return nil, nil
}
func (m *EnhancedMockUserRepository) FindSimilarUsersBySpotifyTopArtists(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]business.SimilarUserResult, error) {
	return []business.SimilarUserResult{}, nil
}

// Track similarity stub to satisfy interface in tests
func (m *EnhancedMockUserRepository) FindSimilarUsersBySpotifyTopTracks(ctx context.Context, anchorUserID uuid.UUID, rng string, limit int32) ([]business.SimilarUserResult, error) {
	return []business.SimilarUserResult{}, nil
}

var _ = Describe("PasswordResetService", func() {
	var (
		service   *business.PasswordResetService
		mockRepo  *EnhancedMockUserRepository
		mockEmail *business.EmailService
		ctx       context.Context
		userID    uuid.UUID
		userEmail string
		testUser  *sqlc.User
	)

	BeforeEach(func() {
		mockRepo = NewEnhancedMockUserRepository()
		mockEmail = business.NewEmailService(&config.EmailConfig{Enabled: false})
		service = business.NewPasswordResetService(mockRepo, mockEmail)
		ctx = context.Background()
		userID = uuid.New()
		userEmail = "test@example.com"
		testUser = &sqlc.User{ID: userID, Username: "testuser", Email: userEmail, FirstName: "Test", LastName: "User", Program: pgtype.Text{String: "2Y", Valid: true}, GraduationYear: pgtype.Int4{Int32: 2026, Valid: true}}
	})

	Describe("ForgotPassword", func() {
		BeforeEach(func() { mockRepo.SetUser(userEmail, testUser) })
		It("creates token and returns 200", func() {
			resp, err := service.ForgotPassword(ctx, generated.ForgotPasswordRequest{Email: userEmail})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(len(mockRepo.passwordResetTokens)).To(Equal(1))
		})

		It("returns 200 even if user not found (security)", func() {
			resp, err := service.ForgotPassword(ctx, generated.ForgotPasswordRequest{Email: "unknown@example.com"})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Code).To(Equal(http.StatusOK))
		})
	})
})
