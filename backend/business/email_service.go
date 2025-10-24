package business

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// EmailService handles sending emails
type EmailService struct {
	config *config.EmailConfig
}

// NewEmailService creates a new email service
func NewEmailService(config *config.EmailConfig) *EmailService {
	return &EmailService{
		config: config,
	}
}

// EmailMessage represents an email to be sent
type EmailMessage struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

// SendEmail sends an email using the configured provider
func (s *EmailService) SendEmail(ctx context.Context, message EmailMessage) error {
	if !s.config.Enabled {
		// In development, just log the email instead of sending
		logger.FromCtx(ctx).Info("email disabled; skipping send", "to", message.To, "subject", message.Subject)
		return nil
	}

	switch strings.ToLower(s.config.Provider) {
	case "sendgrid":
		return s.sendWithSendGrid(ctx, message)
	case "smtp":
		return s.sendWithSMTP(ctx, message)
	default:
		return fmt.Errorf("unsupported email provider: %s", s.config.Provider)
	}
}

// sendWithSendGrid sends email using SendGrid API
func (s *EmailService) sendWithSendGrid(ctx context.Context, message EmailMessage) error {
	if s.config.APIKey == "" {
		return fmt.Errorf("SendGrid API key not configured")
	}

	// Create SendGrid client
	client := sendgrid.NewSendClient(s.config.APIKey)

	// Create email message
	from := mail.NewEmail(s.config.FromName, s.config.FromEmail)
	to := mail.NewEmail("", message.To)

	// Create content
	var content *mail.Content
	if message.IsHTML {
		content = mail.NewContent("text/html", message.Body)
	} else {
		content = mail.NewContent("text/plain", message.Body)
	}

	// Create mail object
	email := mail.NewV3MailInit(from, message.Subject, to, content)

	// Send email
	response, err := client.Send(email)
	if err != nil {
		return fmt.Errorf("failed to send email via SendGrid: %w", err)
	}

	// Check response status
	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid API error: status %d, body: %s", response.StatusCode, response.Body)
	}

	logger.FromCtx(ctx).Info("email sent via sendgrid", "to", message.To, "status", response.StatusCode)
	return nil
}

// sendWithSMTP sends email using SMTP
func (s *EmailService) sendWithSMTP(ctx context.Context, message EmailMessage) error {
	if s.config.SMTPHost == "" || s.config.SMTPPort == "" {
		return fmt.Errorf("SMTP configuration incomplete")
	}

	// Create message
	contentType := "text/plain"
	if message.IsHTML {
		contentType = "text/html"
	}

	msg := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: %s; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n",
		s.config.FromName, s.config.FromEmail,
		message.To,
		message.Subject,
		contentType,
		message.Body)

	// Setup authentication
	auth := smtp.PlainAuth("", s.config.SMTPUser, s.config.SMTPPass, s.config.SMTPHost)

	// Send email
	addr := s.config.SMTPHost + ":" + s.config.SMTPPort
	return smtp.SendMail(addr, auth, s.config.FromEmail, []string{message.To}, []byte(msg))
}

// getEnvWithDefault gets an environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SendPasswordResetEmail sends a password reset email
func (s *EmailService) SendPasswordResetEmail(ctx context.Context, email, username, resetToken string) error {
	// Use environment variable for base URL, fallback to localhost for development
	baseURL := getEnvWithDefault("APP_BASE_URL", "http://localhost:4200")
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, resetToken)

	subject := "Password Reset - Affyne"
	body := fmt.Sprintf(`Hi %s,

You recently requested to reset your password for your Affyne account.

Click the link below to reset your password:
%s

This link will expire in 1 hour for security reasons.

If you didn't request this password reset, please ignore this email or contact support if you have concerns.

Best regards,
The Affyne Team`, username, resetURL)

	message := EmailMessage{
		To:      email,
		Subject: subject,
		Body:    body,
		IsHTML:  false,
	}

	return s.SendEmail(ctx, message)
}
