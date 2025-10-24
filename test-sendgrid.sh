#!/bin/bash

# SendGrid Email Test Script
# This script helps test the email functionality with your SendGrid API key

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔧 SendGrid Email Configuration Test${NC}"
echo "================================================"

# Check if SendGrid API key is provided
if [ -z "$SENDGRID_API_KEY" ]; then
    echo -e "${RED}❌ SENDGRID_API_KEY environment variable not set${NC}"
    echo ""
    echo "Please set your SendGrid API key:"
    echo "export SENDGRID_API_KEY=SG.your_actual_api_key_here"
    echo ""
    echo "Get your API key from: https://app.sendgrid.com/settings/api_keys"
    exit 1
fi

# Validate API key format
if [[ ! "$SENDGRID_API_KEY" =~ ^SG\. ]]; then
    echo -e "${YELLOW}⚠️  Warning: API key should start with 'SG.'${NC}"
fi

echo -e "${GREEN}✅ SendGrid API key found${NC}"

# Set default email configuration
export EMAIL_ENABLED=true
export EMAIL_PROVIDER=sendgrid
export EMAIL_FROM_EMAIL=${EMAIL_FROM_EMAIL:-"noreply@kellogg-music-match.com"}
export EMAIL_FROM_NAME=${EMAIL_FROM_NAME:-"Affyne"}
export APP_BASE_URL=${APP_BASE_URL:-"http://localhost:4200"}

echo ""
echo "Email Configuration:"
echo "  Provider: $EMAIL_PROVIDER"
echo "  From Email: $EMAIL_FROM_EMAIL"
echo "  From Name: $EMAIL_FROM_NAME"
echo "  Base URL: $APP_BASE_URL"
echo "  API Key: ${SENDGRID_API_KEY:0:10}..."
echo ""

# Start the backend
echo -e "${BLUE}🚀 Starting backend with email enabled...${NC}"
cd "$(dirname "$0")/backend"

echo "To test password reset email, use:"
echo ""
echo -e "${YELLOW}curl -X POST http://localhost:8080/auth/forgot-password \\${NC}"
echo -e "${YELLOW}  -H \"Content-Type: application/json\" \\${NC}"
echo -e "${YELLOW}  -d '{\"email\": \"your-email@example.com\"}'${NC}"
echo ""

# Run the backend
go run ./cmd