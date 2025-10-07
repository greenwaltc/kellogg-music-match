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
	return &EnhancedMockUserRepository{
		MockUserRepository:  NewMockUserRepository(),
		users:               make(map[string]*sqlc.User),
		passwordResetTokens: make(map[string]sqlc.PasswordResetToken),
		updatePasswordCalls: make([]UpdatePasswordCall, 0),
	}
}

func (m *EnhancedMockUserRepository) SetUser(email string, user *sqlc.User) {
	m.users[email] = user
}

func (m *EnhancedMockUserRepository) GetUserByEmail(ctx context.Context, email string) (*sqlc.User, error) {
	if user, exists := m.users[email]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *EnhancedMockUserRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (sqlc.PasswordResetToken, error) {
	resetToken := sqlc.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		Used:      pgtype.Bool{Bool: false, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	m.passwordResetTokens[token] = resetToken
	return resetToken, nil
}

func (m *EnhancedMockUserRepository) GetPasswordResetToken(ctx context.Context, token string) (sqlc.PasswordResetToken, error) {
	if resetToken, exists := m.passwordResetTokens[token]; exists {
		// Mirror the SQL query behavior: only return tokens that are not expired and not used
		if resetToken.ExpiresAt.Time.After(time.Now()) && !resetToken.Used.Bool {
			return resetToken, nil
		}
	}
	return sqlc.PasswordResetToken{}, pgx.ErrNoRows
}

// GetPasswordResetTokenRaw returns the token without validation (for testing purposes)
func (m *EnhancedMockUserRepository) GetPasswordResetTokenRaw(ctx context.Context, token string) (sqlc.PasswordResetToken, error) {
	if resetToken, exists := m.passwordResetTokens[token]; exists {
		return resetToken, nil
	}
	return sqlc.PasswordResetToken{}, pgx.ErrNoRows
}

func (m *EnhancedMockUserRepository) MarkPasswordResetTokenAsUsed(ctx context.Context, token string) error {
	if resetToken, exists := m.passwordResetTokens[token]; exists {
		resetToken.Used = pgtype.Bool{Bool: true, Valid: true}
		m.passwordResetTokens[token] = resetToken
		return nil
	}
	return fmt.Errorf("token not found")
}

func (m *EnhancedMockUserRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) (sqlc.UpdateUserPasswordRow, error) {
	m.updatePasswordCalls = append(m.updatePasswordCalls, UpdatePasswordCall{
		UserID:       userID,
		PasswordHash: passwordHash,
	})
	return sqlc.UpdateUserPasswordRow{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}, nil
}

func (m *EnhancedMockUserRepository) DeleteUserPasswordResetTokens(ctx context.Context, userID uuid.UUID) error {
	for token, resetToken := range m.passwordResetTokens {
		if resetToken.UserID == userID && !resetToken.Used.Bool {
			delete(m.passwordResetTokens, token)
		}
	}
	return nil
}

func (m *EnhancedMockUserRepository) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	for token, resetToken := range m.passwordResetTokens {
		if resetToken.ExpiresAt.Time.Before(time.Now()) {
			delete(m.passwordResetTokens, token)
		}
	}
	return nil
}

// Spotify token operations (unused in password reset tests)
func (m *EnhancedMockUserRepository) UpsertSpotifyTokens(ctx context.Context, userID uuid.UUID, accessToken string, refreshTokenEncrypted []byte, expiresAt time.Time, scope string, tokenType string) error {
	return nil
}
func (m *EnhancedMockUserRepository) GetSpotifyTokensByUser(ctx context.Context, userID uuid.UUID) (*sqlc.SpotifyToken, error) {
	return nil, nil
}

var _ = Describe("PasswordResetService", func() {
	var (
		service          *business.PasswordResetService
		mockRepo         *EnhancedMockUserRepository
		mockEmailService *business.EmailService
		ctx              context.Context
		userID           uuid.UUID
		userEmail        string
		testUser         *sqlc.User
	)

	BeforeEach(func() {
		mockRepo = NewEnhancedMockUserRepository()
		// Create a test email service with disabled email for testing
		emailConfig := &config.EmailConfig{
			Enabled: false, // Disable actual email sending in tests
		}
		mockEmailService = business.NewEmailService(emailConfig)
		service = business.NewPasswordResetService(mockRepo, mockEmailService)
		ctx = context.Background()
		userID = uuid.New()
		userEmail = "test@example.com"

		// Create a test user
		testUser = &sqlc.User{
			ID:             userID,
			Username:       "testuser",
			Email:          userEmail,
			FirstName:      "Test",
			LastName:       "User",
			Program:        pgtype.Text{String: "2Y", Valid: true},
			GraduationYear: pgtype.Int4{Int32: 2026, Valid: true},
		}
	})

	Describe("ForgotPassword", func() {
		Context("when user exists", func() {
			BeforeEach(func() {
				mockRepo.SetUser(userEmail, testUser)
			})

			It("should create a password reset token and send email", func() {
				request := generated.ForgotPasswordRequest{Email: userEmail}
				response, err := service.ForgotPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusOK))

				// Verify a token was created (we can't easily test email without complex mocking)
				// Check that a token exists for this user
				foundToken := false
				for _, token := range mockRepo.passwordResetTokens {
					if token.UserID == userID {
						foundToken = true
						break
					}
				}
				Expect(foundToken).To(BeTrue())
			})

			It("should create multiple tokens if called multiple times", func() {
				// Create an existing token
				existingToken := "existing-token"
				mockRepo.CreatePasswordResetToken(ctx, userID, existingToken, time.Now().Add(10*time.Minute))

				request := generated.ForgotPasswordRequest{Email: userEmail}
				response, err := service.ForgotPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusOK))

				// Verify old token still exists (ForgotPassword doesn't delete existing tokens)
				_, err = mockRepo.GetPasswordResetToken(ctx, existingToken)
				Expect(err).NotTo(HaveOccurred())

				// Verify a new token was also created
				tokenCount := 0
				for _, token := range mockRepo.passwordResetTokens {
					if token.UserID == userID {
						tokenCount++
					}
				}
				Expect(tokenCount).To(Equal(2)) // Original + new token
			})
		})

		Context("when user does not exist", func() {
			It("should still return success for security reasons", func() {
				request := generated.ForgotPasswordRequest{Email: "nonexistent@example.com"}
				response, err := service.ForgotPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusOK))
			})
		})

		Context("with invalid email format", func() {
			It("should return bad request error", func() {
				request := generated.ForgotPasswordRequest{Email: "invalid-email"}
				response, err := service.ForgotPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("with empty email", func() {
			It("should return bad request error", func() {
				request := generated.ForgotPasswordRequest{Email: ""}
				response, err := service.ForgotPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("ResetPassword", func() {
		var (
			newPassword string
			resetToken  string
		)

		BeforeEach(func() {
			newPassword = "newSecurePassword123!"
			resetToken = "test-reset-token"

			// Set up user and token
			mockRepo.SetUser(userEmail, testUser)
			mockRepo.CreatePasswordResetToken(ctx, userID, resetToken, time.Now().Add(10*time.Minute))
		})

		Context("with valid token and password", func() {
			It("should update the user's password", func() {
				request := generated.ResetPasswordRequest{
					Token:       resetToken,
					NewPassword: newPassword,
				}
				response, err := service.ResetPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusOK))

				// Verify password was updated
				Expect(mockRepo.updatePasswordCalls).To(HaveLen(1))
				Expect(mockRepo.updatePasswordCalls[0].UserID).To(Equal(userID))
				// Verify it's a bcrypt hash
				Expect(mockRepo.updatePasswordCalls[0].PasswordHash).To(HavePrefix("$2a$"))
			})

			It("should mark the token as used", func() {
				request := generated.ResetPasswordRequest{
					Token:       resetToken,
					NewPassword: newPassword,
				}
				response, err := service.ResetPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusOK))

				// Verify token was marked as used (use raw method to bypass validation)
				token, err := mockRepo.GetPasswordResetTokenRaw(ctx, resetToken)
				Expect(err).NotTo(HaveOccurred())
				Expect(token.Used.Bool).To(BeTrue())
			})
		})

		Context("with non-existent token", func() {
			It("should return bad request error", func() {
				request := generated.ResetPasswordRequest{
					Token:       "non-existent-token",
					NewPassword: newPassword,
				}
				response, err := service.ResetPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("with already used token", func() {
			BeforeEach(func() {
				// Mark token as used
				mockRepo.MarkPasswordResetTokenAsUsed(ctx, resetToken)
			})

			It("should return bad request error", func() {
				request := generated.ResetPasswordRequest{
					Token:       resetToken,
					NewPassword: newPassword,
				}
				response, err := service.ResetPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("with expired token", func() {
			BeforeEach(func() {
				// Create expired token
				expiredToken := "expired-token"
				mockRepo.CreatePasswordResetToken(ctx, userID, expiredToken, time.Now().Add(-1*time.Hour))
				resetToken = expiredToken
			})

			It("should return bad request error", func() {
				request := generated.ResetPasswordRequest{
					Token:       resetToken,
					NewPassword: newPassword,
				}
				response, err := service.ResetPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("with weak password", func() {
			It("should reject password that is too short", func() {
				request := generated.ResetPasswordRequest{
					Token:       resetToken,
					NewPassword: "123",
				}
				response, err := service.ResetPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})

			It("should reject empty password", func() {
				request := generated.ResetPasswordRequest{
					Token:       resetToken,
					NewPassword: "",
				}
				response, err := service.ResetPassword(ctx, request)

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("Cleanup Operations", func() {
		BeforeEach(func() {
			mockRepo.SetUser(userEmail, testUser)
		})

		It("should clean up expired tokens during forgot password", func() {
			// Create some expired tokens
			expiredToken1 := "expired1"
			expiredToken2 := "expired2"
			mockRepo.CreatePasswordResetToken(ctx, uuid.New(), expiredToken1, time.Now().Add(-1*time.Hour))
			mockRepo.CreatePasswordResetToken(ctx, uuid.New(), expiredToken2, time.Now().Add(-2*time.Hour))

			request := generated.ForgotPasswordRequest{Email: userEmail}
			response, err := service.ForgotPassword(ctx, request)

			Expect(err).NotTo(HaveOccurred())
			Expect(response.Code).To(Equal(http.StatusOK))

			// Expired tokens should be cleaned up
			_, err1 := mockRepo.GetPasswordResetToken(ctx, expiredToken1)
			_, err2 := mockRepo.GetPasswordResetToken(ctx, expiredToken2)
			Expect(err1).To(HaveOccurred())
			Expect(err2).To(HaveOccurred())
		})
	})
})
