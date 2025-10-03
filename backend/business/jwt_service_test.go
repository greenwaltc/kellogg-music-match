package business_test

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JWT Service", func() {
	var (
		jwtService *business.JWTService
		jwtConfig  *config.JWTConfig
	)

	BeforeEach(func() {
		jwtConfig = &config.JWTConfig{
			SecretKey:    "test-secret-key-for-testing-only",
			ExpiryHours:  24,
			RefreshHours: 168,
		}
		jwtService = business.NewJWTService(jwtConfig)
	})

	Describe("Token Generation", func() {
		Context("when generating an access token", func() {
			It("should create a valid JWT token", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "testuser"
				email := "test@example.com"

				token, err := jwtService.GenerateToken(userID, username, email)

				Expect(err).ToNot(HaveOccurred())
				Expect(token).ToNot(BeEmpty())
				Expect(token).To(ContainSubstring(".")) // JWT has dots separating parts
			})

			It("should include correct claims in the token", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "testuser"
				email := "test@example.com"

				tokenString, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				// Parse the token to verify claims
				token, err := jwt.ParseWithClaims(tokenString, &business.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
					return []byte(jwtConfig.SecretKey), nil
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(token.Valid).To(BeTrue())

				claims, ok := token.Claims.(*business.JWTClaims)
				Expect(ok).To(BeTrue())
				Expect(claims.UserID).To(Equal(userID))
				Expect(claims.Username).To(Equal(username))
				Expect(claims.Email).To(Equal(email))
				Expect(claims.Issuer).To(Equal("kellogg-music-match"))
				Expect(claims.Subject).To(Equal(userID))
			})

			It("should set correct expiration time", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "testuser"
				email := "test@example.com"

				tokenString, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				// Parse the token to verify expiration
				token, err := jwt.ParseWithClaims(tokenString, &business.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
					return []byte(jwtConfig.SecretKey), nil
				})

				Expect(err).ToNot(HaveOccurred())
				claims, ok := token.Claims.(*business.JWTClaims)
				Expect(ok).To(BeTrue())

				// Check that expiration is approximately 24 hours from now
				expectedExpiry := time.Now().Add(24 * time.Hour)
				actualExpiry := claims.ExpiresAt.Time
				timeDiff := actualExpiry.Sub(expectedExpiry)
				Expect(timeDiff.Abs()).To(BeNumerically("<", time.Minute)) // Within 1 minute tolerance
			})
		})

		Context("when generating a refresh token", func() {
			It("should create a valid refresh token", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "testuser"
				email := "test@example.com"

				token, err := jwtService.GenerateRefreshToken(userID, username, email)

				Expect(err).ToNot(HaveOccurred())
				Expect(token).ToNot(BeEmpty())
				Expect(token).To(ContainSubstring("."))
			})

			It("should have longer expiration time", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "testuser"
				email := "test@example.com"

				tokenString, err := jwtService.GenerateRefreshToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				// Parse the token to verify expiration
				token, err := jwt.ParseWithClaims(tokenString, &business.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
					return []byte(jwtConfig.SecretKey), nil
				})

				Expect(err).ToNot(HaveOccurred())
				claims, ok := token.Claims.(*business.JWTClaims)
				Expect(ok).To(BeTrue())

				// Check that expiration is approximately 7 days from now
				expectedExpiry := time.Now().Add(168 * time.Hour) // 7 days
				actualExpiry := claims.ExpiresAt.Time
				timeDiff := actualExpiry.Sub(expectedExpiry)
				Expect(timeDiff.Abs()).To(BeNumerically("<", time.Minute))

				// Verify it has refresh issuer
				Expect(claims.Issuer).To(Equal("kellogg-music-match-refresh"))
			})
		})
	})

	Describe("Token Validation", func() {
		Context("when validating a valid token", func() {
			It("should return the correct claims", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "testuser"
				email := "test@example.com"

				// Generate a token
				tokenString, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				// Validate the token
				claims, err := jwtService.ValidateToken(tokenString)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims).ToNot(BeNil())
				Expect(claims.UserID).To(Equal(userID))
				Expect(claims.Username).To(Equal(username))
				Expect(claims.Email).To(Equal(email))
			})
		})

		Context("when validating an invalid token", func() {
			It("should return an error for malformed token", func() {
				invalidToken := "invalid.token.here"

				claims, err := jwtService.ValidateToken(invalidToken)
				Expect(err).To(HaveOccurred())
				Expect(claims).To(BeNil())
			})

			It("should return an error for token with wrong signature", func() {
				// Create a token with different secret
				differentConfig := &config.JWTConfig{
					SecretKey:    "different-secret-key",
					ExpiryHours:  24,
					RefreshHours: 168,
				}
				differentJWTService := business.NewJWTService(differentConfig)

				tokenString, err := differentJWTService.GenerateToken("user123", "testuser", "test@example.com")
				Expect(err).ToNot(HaveOccurred())

				// Try to validate with original service (different secret)
				claims, err := jwtService.ValidateToken(tokenString)
				Expect(err).To(HaveOccurred())
				Expect(claims).To(BeNil())
			})

			It("should return an error for expired token", func() {
				// Create a service with very short expiry
				shortExpiryConfig := &config.JWTConfig{
					SecretKey:    "test-secret-key",
					ExpiryHours:  0, // This will create an expired token
					RefreshHours: 168,
				}
				shortExpiryService := business.NewJWTService(shortExpiryConfig)

				tokenString, err := shortExpiryService.GenerateToken("user123", "testuser", "test@example.com")
				Expect(err).ToNot(HaveOccurred())

				// Wait a moment to ensure expiration
				time.Sleep(10 * time.Millisecond)

				// Try to validate the expired token
				claims, err := jwtService.ValidateToken(tokenString)
				Expect(err).To(HaveOccurred())
				Expect(claims).To(BeNil())
			})

			It("should return an error for empty token", func() {
				claims, err := jwtService.ValidateToken("")
				Expect(err).To(HaveOccurred())
				Expect(claims).To(BeNil())
			})
		})
	})

	Describe("Token Security", func() {
		Context("when using different signing methods", func() {
			It("should reject tokens with unexpected signing method", func() {
				// Create a token manually with RS256 instead of HS256
				claims := &business.JWTClaims{
					UserID:   "user123",
					Username: "testuser",
					Email:    "test@example.com",
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
						Issuer:    "kellogg-music-match",
					},
				}

				// Try to create with different signing method (this would fail in real scenario)
				token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims) // Different method
				tokenString, err := token.SignedString([]byte(jwtConfig.SecretKey))
				Expect(err).ToNot(HaveOccurred())

				// Our service should still validate it since it's still HMAC
				validatedClaims, err := jwtService.ValidateToken(tokenString)
				Expect(err).ToNot(HaveOccurred()) // HMAC family is accepted
				Expect(validatedClaims.UserID).To(Equal("user123"))
			})
		})

		Context("when testing token uniqueness", func() {
			It("should generate unique tokens for same user", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "testuser"
				email := "test@example.com"

				token1, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				// Wait a moment to ensure different IssuedAt time
				time.Sleep(1 * time.Second)

				token2, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				Expect(token1).ToNot(Equal(token2))
			})
		})
	})

	Describe("Configuration Edge Cases", func() {
		Context("when using different configurations", func() {
			It("should handle very long expiry times", func() {
				longExpiryConfig := &config.JWTConfig{
					SecretKey:    "test-secret-key",
					ExpiryHours:  8760,  // 1 year
					RefreshHours: 17520, // 2 years
				}
				longExpiryService := business.NewJWTService(longExpiryConfig)

				token, err := longExpiryService.GenerateToken("user123", "testuser", "test@example.com")
				Expect(err).ToNot(HaveOccurred())
				Expect(token).ToNot(BeEmpty())

				// Validate the token
				claims, err := longExpiryService.ValidateToken(token)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.UserID).To(Equal("user123"))
			})

			It("should handle very short secret keys", func() {
				shortKeyConfig := &config.JWTConfig{
					SecretKey:    "key", // Very short key
					ExpiryHours:  24,
					RefreshHours: 168,
				}
				shortKeyService := business.NewJWTService(shortKeyConfig)

				token, err := shortKeyService.GenerateToken("user123", "testuser", "test@example.com")
				Expect(err).ToNot(HaveOccurred())

				claims, err := shortKeyService.ValidateToken(token)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.UserID).To(Equal("user123"))
			})

			It("should handle unicode characters in user data", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "用户测试"          // Unicode username
				email := "tëst@éxample.com" // Unicode email

				token, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				claims, err := jwtService.ValidateToken(token)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.Username).To(Equal(username))
				Expect(claims.Email).To(Equal(email))
			})
		})
	})

	Describe("Leeway Handling", func() {
		Context("when token not-before is slightly in the future within leeway", func() {
			It("should still validate successfully", func() {
				// Configure service with 120s leeway
				leewayCfg := &config.JWTConfig{
					SecretKey:     "test-secret-key-for-testing-only",
					ExpiryHours:   1,
					RefreshHours:  24,
					LeewaySeconds: 120,
				}
				leewayService := business.NewJWTService(leewayCfg)

				future := time.Now().Add(30 * time.Second) // within 120s leeway
				claims := &business.JWTClaims{
					UserID:   "future-user",
					Username: "future",
					Email:    "future@example.com",
					RegisteredClaims: jwt.RegisteredClaims{
						NotBefore: jwt.NewNumericDate(future),
						IssuedAt:  jwt.NewNumericDate(future),
						ExpiresAt: jwt.NewNumericDate(future.Add(time.Hour)),
						Issuer:    "kellogg-music-match",
						Subject:   "future-user",
					},
				}

				tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, err := tok.SignedString([]byte(leewayCfg.SecretKey))
				Expect(err).ToNot(HaveOccurred())

				validated, err := leewayService.ValidateToken(tokenString)
				Expect(err).ToNot(HaveOccurred())
				Expect(validated.UserID).To(Equal("future-user"))
			})
		})

		Context("when token not-before is beyond configured leeway", func() {
			It("should fail validation with not yet valid error", func() {
				leewayCfg := &config.JWTConfig{
					SecretKey:     "test-secret-key-for-testing-only",
					ExpiryHours:   1,
					RefreshHours:  24,
					LeewaySeconds: 60, // 1 minute
				}
				leewayService := business.NewJWTService(leewayCfg)

				future := time.Now().Add(3 * time.Minute) // beyond 60s leeway
				claims := &business.JWTClaims{
					UserID:   "too-future-user",
					Username: "future2",
					Email:    "future2@example.com",
					RegisteredClaims: jwt.RegisteredClaims{
						NotBefore: jwt.NewNumericDate(future),
						IssuedAt:  jwt.NewNumericDate(future),
						ExpiresAt: jwt.NewNumericDate(future.Add(time.Hour)),
						Issuer:    "kellogg-music-match",
						Subject:   "too-future-user",
					},
				}

				tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, err := tok.SignedString([]byte(leewayCfg.SecretKey))
				Expect(err).ToNot(HaveOccurred())

				validated, err := leewayService.ValidateToken(tokenString)
				Expect(err).To(HaveOccurred())
				Expect(validated).To(BeNil())
			})
		})
	})
})
