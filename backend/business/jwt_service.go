package business

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
)

// JWTClaims represents the claims for JWT tokens
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token generation and validation
type JWTService struct {
	config *config.JWTConfig
}

// NewJWTService creates a new JWT service
func NewJWTService(config *config.JWTConfig) *JWTService {
	return &JWTService{
		config: config,
	}
}

// GenerateToken generates a new JWT token for a user
func (s *JWTService) GenerateToken(userID, username, email string) (string, error) {
	// Create the claims
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.config.ExpiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "kellogg-music-match",
			Subject:   userID,
		},
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret
	tokenString, err := token.SignedString([]byte(s.config.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	// Build a parser with configured leeway to tolerate minor clock skew
	parser := jwt.NewParser(jwt.WithLeeway(time.Duration(s.config.LeewaySeconds) * time.Second))

	// Parse the token with typed claims
	token, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.config.SecretKey), nil
	})

	if err != nil {
		logJWTError("parse_error", tokenString, err)
		return nil, err
	}

	// Check if token is valid
	if !token.Valid {
		logJWTError("invalid_token", tokenString, errors.New("token not valid"))
		return nil, errors.New("token is not valid")
	}

	// Extract claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, errors.New("could not parse claims")
	}

	// Legacy fallback: if user_id missing but camelCase userId present.
	// Because we parsed into a typed struct, token.Claims is *JWTClaims, so we re-parse
	// into MapClaims only when needed to avoid performance hit on modern tokens.
	if claims.UserID == "" {
		legacyToken, err2 := parser.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			// Re-validate signing method & secret
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(s.config.SecretKey), nil
		})
		if err2 == nil && legacyToken != nil && legacyToken.Valid {
			if mapClaims, ok := legacyToken.Claims.(jwt.MapClaims); ok {
				if v, exists := mapClaims["userId"].(string); exists && v != "" {
					claims.UserID = v
					// Populate username/email if somehow absent (should already be set)
					if claims.Username == "" {
						if u, ok := mapClaims["username"].(string); ok {
							claims.Username = u
						}
					}
					if claims.Email == "" {
						if e, ok := mapClaims["email"].(string); ok {
							claims.Email = e
						}
					}
				}
			}
		} else if err2 != nil {
			logJWTError("legacy_parse_error", tokenString, err2)
		}
	}

	return claims, nil
}

// logJWTError emits structured debug logs for JWT validation issues without leaking full token contents.
func logJWTError(reason string, tokenString string, err error) {
	if tokenString == "" {
		logger.L().Debug("jwt validation", "reason", reason, "error", err)
		return
	}
	// Hash the token to correlate without exposure
	sum := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(sum[:8]) // first 8 bytes for brevity
	// Classify known validation errors
	classification := reason
	if errors.Is(err, jwt.ErrTokenExpired) {
		classification = "expired"
	} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
		classification = "not_yet_valid"
	} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		classification = "bad_signature"
	}
	logger.L().Debug("jwt validation", "reason", classification, "error", err, "tokenHash", tokenHash)
}

// GenerateRefreshToken generates a longer-lived refresh token
func (s *JWTService) GenerateRefreshToken(userID, username, email string) (string, error) {
	// Create the claims with longer expiry
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.config.RefreshHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "kellogg-music-match-refresh",
			Subject:   userID,
		},
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret
	tokenString, err := token.SignedString([]byte(s.config.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
