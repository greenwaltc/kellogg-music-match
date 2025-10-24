# SendGrid Email Configuration Guide

This guide will help you configure your Affyne application to send password reset emails using SendGrid.

## ✅ Implementation Complete

The application now has full SendGrid integration implemented:

- ✅ SendGrid Go library integrated
- ✅ Email service with SendGrid API implementation
- ✅ Environment variable configuration
- ✅ Password reset email templates
- ✅ Pulumi deployment configuration
- ✅ Error handling and logging

## 🔧 Configuration Steps

### 1. Update Pulumi Configuration

Replace the placeholder SendGrid API key in `/pulumi/main.go`:

```go
// Find this line (around line 483):
&corev1.EnvVarArgs{
    Name:  pulumi.String("SENDGRID_API_KEY"),
    Value: pulumi.String("dummy-key-replace-when-configuring-sendgrid"), // ← Replace this
},

// Replace with your actual SendGrid API key:
&corev1.EnvVarArgs{
    Name:  pulumi.String("SENDGRID_API_KEY"),
    Value: pulumi.String("SG.your_actual_sendgrid_api_key_here"), // ← Your key here
},
```

### 2. Update Email Configuration (Optional)

You can also update the "from" email address in Pulumi:

```go
&corev1.EnvVarArgs{
    Name:  pulumi.String("EMAIL_FROM_EMAIL"),
    Value: pulumi.String("noreply@yourdomain.com"), // ← Your verified sender email
},
```

### 3. Local Development Configuration

For local development, set these environment variables:

```bash
# In your shell or .env file:
export EMAIL_ENABLED=true
export EMAIL_PROVIDER=sendgrid
export SENDGRID_API_KEY=SG.your_actual_sendgrid_api_key_here
export EMAIL_FROM_EMAIL=noreply@yourdomain.com
export EMAIL_FROM_NAME="Affyne"
export APP_BASE_URL=http://localhost:4200
```

### 4. Docker Compose Configuration

For Docker development, add these to your `docker-compose.yml` backend service:

```yaml
services:
  backend:
    environment:
      - EMAIL_ENABLED=true
      - EMAIL_PROVIDER=sendgrid
      - SENDGRID_API_KEY=SG.your_actual_sendgrid_api_key_here
      - EMAIL_FROM_EMAIL=noreply@yourdomain.com
      - EMAIL_FROM_NAME=Affyne
      - APP_BASE_URL=http://localhost:4200
```

## 🔒 SendGrid Setup Requirements

### 1. SendGrid Account Setup
1. Create account at [sendgrid.com](https://sendgrid.com)
2. Complete account verification
3. Set up domain authentication (recommended for production)

### 2. API Key Creation
1. Go to Settings → API Keys in SendGrid dashboard
2. Create a new API Key with "Mail Send" permissions
3. Copy the API key (starts with `SG.`)

### 3. Sender Identity Verification
- **Single Sender Verification**: Verify individual email addresses
- **Domain Authentication**: Verify entire domain (recommended for production)

## 🧪 Testing

### Test Email Sending (Development)
```bash
# Start your backend with email enabled
EMAIL_ENABLED=true SENDGRID_API_KEY=your_key go run ./cmd

# Test password reset through the UI or API
curl -X POST http://localhost:8080/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com"}'
```

### Debugging
The application logs email sending status:
- Success: `"Email sent successfully via SendGrid to email@example.com (status: 202)"`
- Error: Detailed error messages with status codes and responses

## 🚀 Deployment

### Kubernetes/Pulumi Deployment
```bash
cd pulumi
pulumi up
```

The deployment will automatically use the configured environment variables.

### Production Considerations
1. **Domain Authentication**: Set up domain authentication in SendGrid for better deliverability
2. **Rate Limits**: SendGrid has rate limits - monitor usage
3. **Monitoring**: Set up alerts for email delivery failures
4. **Security**: Store API keys as Kubernetes secrets in production (not hardcoded)

## 🔍 Environment Variables Reference

| Variable | Description | Example |
|----------|-------------|---------|
| `EMAIL_ENABLED` | Enable/disable email sending | `true` |
| `EMAIL_PROVIDER` | Email service provider | `sendgrid` |
| `SENDGRID_API_KEY` | SendGrid API key | `SG.abc123...` |
| `EMAIL_FROM_EMAIL` | Sender email address | `noreply@yourdomain.com` |
| `EMAIL_FROM_NAME` | Sender display name | `Affyne` |
| `APP_BASE_URL` | Base URL for reset links | `https://yourdomain.com` |

## 🐛 Troubleshooting

### Common Issues

1. **"SendGrid API key not configured"**
   - Ensure `SENDGRID_API_KEY` environment variable is set
   - Verify the API key is correct and has Mail Send permissions

2. **"SendGrid API error: status 401"**
   - API key is invalid or has insufficient permissions
   - Check the API key in SendGrid dashboard

3. **"SendGrid API error: status 403"**
   - Sender email not verified
   - Complete sender verification in SendGrid

4. **Emails not being received**
   - Check spam folders
   - Verify sender email authentication
   - Check SendGrid activity logs

### Verification Steps
1. Check backend logs for email sending status
2. Verify environment variables are loaded: check `/health` endpoint
3. Test with a verified email address first
4. Check SendGrid dashboard for email activity

## 📧 Email Template

The current password reset email template includes:
- Personalized greeting with username
- Secure reset link with token
- 1-hour expiration notice
- Clear instructions
- Professional signature

The template can be customized in `/backend/business/email_service.go` in the `SendPasswordResetEmail` function.