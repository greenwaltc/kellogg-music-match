package business

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// PasswordResetService handles password reset functionality
type PasswordResetService struct {
	userRepo     UserRepository
	emailService *EmailService
}

// NewPasswordResetService creates a new password reset service
func NewPasswordResetService(userRepo UserRepository, emailService *EmailService) *PasswordResetService {
	return &PasswordResetService{
		userRepo:     userRepo,
		emailService: emailService,
	}
}

// ForgotPassword initiates the password reset process
func (s *PasswordResetService) ForgotPassword(ctx context.Context, request generated.ForgotPasswordRequest) (generated.ImplResponse, error) {
	email := request.Email
	if email == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "email is required",
		}), nil
	}

	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "invalid email format",
		}), nil
	}

	// Check if user exists (don't reveal if they exist for security)
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Log error but don't reveal to user
		logger.FromCtx(ctx).Error("get user by email failed", "error", err)
	}

	// Always return success for security (don't reveal if email exists)
	if user != nil {
		// Generate reset token
		token, err := generateResetToken()
		if err != nil {
			return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
				Message: "failed to generate reset token",
			}), nil
		}

		// Store token in database (expires in 1 hour)
		expiresAt := time.Now().Add(time.Hour)
		_, err = s.userRepo.CreatePasswordResetToken(ctx, user.ID, token, expiresAt)
		if err != nil {
			logger.FromCtx(ctx).Error("create password reset token failed", "error", err)
			return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
				Message: "failed to create reset token",
			}), nil
		}

		// Send reset email
		err = s.emailService.SendPasswordResetEmail(ctx, email, user.Username, token)
		if err != nil {
			// Log error but don't fail the request
			logger.FromCtx(ctx).Error("send password reset email failed", "error", err)
		}
	}

	return generated.Response(http.StatusOK, generated.MessageResponse{
		Message: "If an account with that email exists, a password reset link has been sent",
	}), nil
}

// ResetPassword resets the user's password using a valid token
func (s *PasswordResetService) ResetPassword(ctx context.Context, request generated.ResetPasswordRequest) (generated.ImplResponse, error) {
	token := request.Token
	newPassword := request.NewPassword

	if token == "" || newPassword == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "token and newPassword are required",
		}), nil
	}

	// Validate password strength
	if !isValidPassword(newPassword) {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "password must be at least 8 characters and contain uppercase, lowercase, and number",
		}), nil
	}

	// Get and validate token
	resetToken, err := s.userRepo.GetPasswordResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
				Message: "invalid or expired reset token",
			}), nil
		}
		logger.FromCtx(ctx).Error("retrieve reset token failed", "error", err)
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to validate reset token",
		}), nil
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to process password",
		}), nil
	}

	// Update user password
	_, err = s.userRepo.UpdateUserPassword(ctx, resetToken.UserID, string(hashedPassword))
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to update password",
		}), nil
	}

	// Mark token as used
	err = s.userRepo.MarkPasswordResetTokenAsUsed(ctx, token)
	if err != nil {
		// Log error but don't fail since password was already updated
		logger.FromCtx(ctx).Error("mark reset token used failed", "error", err)
	}

	// Delete all other reset tokens for this user
	err = s.userRepo.DeleteUserPasswordResetTokens(ctx, resetToken.UserID)
	if err != nil {
		// Log error but don't fail
		logger.FromCtx(ctx).Error("delete other reset tokens failed", "error", err)
	}

	return generated.Response(http.StatusOK, generated.MessageResponse{
		Message: "Password has been reset successfully",
	}), nil
}

// generateResetToken generates a cryptographically secure random token
func generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// isValidPassword validates password strength
func isValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	return hasUpper && hasLower && hasNumber
}
