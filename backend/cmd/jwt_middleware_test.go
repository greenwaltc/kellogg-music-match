package main

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/golang-jwt/jwt/v5"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JWT Middleware", func() {
	var (
		middleware   *JWTMiddleware
		jwtService   *business.JWTService
		nextHandler  http.Handler
		recorder     *httptest.ResponseRecorder
		nextCalled   bool
		capturedUser *UserContext
	)

	BeforeEach(func() {
		jwtConfig := &config.JWTConfig{
			SecretKey:    "test-secret-key-for-middleware-testing",
			ExpiryHours:  24,
			RefreshHours: 168,
		}
		jwtService = business.NewJWTService(jwtConfig)
		middleware = NewJWTMiddleware(jwtService)

		// Create a mock next handler
		nextCalled = false
		capturedUser = nil
		nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			// Capture user context if present
			if user, ok := GetUserFromContext(r.Context()); ok {
				capturedUser = user
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		recorder = httptest.NewRecorder()
	})

	Describe("Public Endpoints", func() {
		Context("when accessing public endpoints", func() {
			It("should allow access to /login without authentication", func() {
				req := httptest.NewRequest("POST", "/login", nil)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(BeTrue())
				Expect(capturedUser).To(BeNil())
			})

			It("should allow access to /register without authentication", func() {
				req := httptest.NewRequest("POST", "/register", nil)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(BeTrue())
				Expect(capturedUser).To(BeNil())
			})

			It("should allow access to /health without authentication", func() {
				req := httptest.NewRequest("GET", "/health", nil)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(BeTrue())
				Expect(capturedUser).To(BeNil())
			})
		})
	})

	Describe("Protected Endpoints with JWT", func() {
		Context("when accessing protected endpoints with valid JWT token", func() {
			It("should allow access and populate user context", func() {
				// Generate a valid token
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "testuser"
				email := "test@example.com"

				token, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				// Create request with Authorization header
				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Bearer "+token)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(BeTrue())
				Expect(capturedUser).ToNot(BeNil())
				Expect(capturedUser.UserID).To(Equal(userID))
				Expect(capturedUser.Username).To(Equal(username))
				Expect(capturedUser.Email).To(Equal(email))
			})

			It("should handle tokens with special characters in user data", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "用户@test.com"
				email := "tëst@éxample.com"

				token, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Bearer "+token)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(BeTrue())
				Expect(capturedUser.Username).To(Equal(username))
				Expect(capturedUser.Email).To(Equal(email))
			})
		})

		Context("when accessing protected endpoints without authorization", func() {
			It("should return 401 when no authorization header is present", func() {
				req := httptest.NewRequest("POST", "/findMusicMatches", nil)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
				Expect(recorder.Body.String()).To(ContainSubstring("Authorization header required"))
			})

			It("should return 401 when authorization header is empty", func() {
				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "")

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
			})
		})

		Context("when accessing protected endpoints with invalid JWT token", func() {
			It("should return 401 for malformed Bearer token", func() {
				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Bearer invalid.token.here")

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
				Expect(recorder.Body.String()).To(ContainSubstring("Invalid token"))
			})

			It("should return 401 for non-Bearer authorization", func() {
				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // Basic auth

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
				Expect(recorder.Body.String()).To(ContainSubstring("Bearer token required"))
			})

			It("should return 401 for expired token", func() {
				// Create a service with 0 expiry to generate expired token
				expiredConfig := &config.JWTConfig{
					SecretKey:    "test-secret-key-for-middleware-testing",
					ExpiryHours:  0,
					RefreshHours: 168,
				}
				expiredJWTService := business.NewJWTService(expiredConfig)

				token, err := expiredJWTService.GenerateToken("user123", "testuser", "test@example.com")
				Expect(err).ToNot(HaveOccurred())

				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Bearer "+token)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
				Expect(recorder.Body.String()).To(ContainSubstring("Invalid token"))
			})

			It("should return 401 for token with wrong signature", func() {
				// Create token with different secret
				differentConfig := &config.JWTConfig{
					SecretKey:    "different-secret-key",
					ExpiryHours:  24,
					RefreshHours: 168,
				}
				differentJWTService := business.NewJWTService(differentConfig)

				token, err := differentJWTService.GenerateToken("user123", "testuser", "test@example.com")
				Expect(err).ToNot(HaveOccurred())

				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Bearer "+token)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
			})
		})
	})

	Describe("Backward Compatibility", func() {
		Context("when using legacy X-User-Username header", func() {
			It("should allow access with X-User-Username header", func() {
				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("X-User-Username", "legacyuser")

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(BeTrue())
				Expect(capturedUser).ToNot(BeNil())
				Expect(capturedUser.Username).To(Equal("legacyuser"))
				Expect(capturedUser.UserID).To(BeEmpty()) // No user ID in legacy mode
				Expect(capturedUser.Email).To(BeEmpty())  // No email in legacy mode
			})

			It("should prefer JWT token over X-User-Username when both are present", func() {
				// Generate a valid JWT token
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "jwtuser"
				email := "jwt@example.com"

				token, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("X-User-Username", "legacyuser") // This should be ignored

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(BeTrue())
				Expect(capturedUser).ToNot(BeNil())
				Expect(capturedUser.Username).To(Equal("jwtuser")) // JWT user, not legacy
				Expect(capturedUser.UserID).To(Equal(userID))
				Expect(capturedUser.Email).To(Equal(email))
			})
		})

		Context("when using legacy camelCase userId in token", func() {
			It("should still populate UserID via fallback", func() {
				userID := "123e4567-e89b-12d3-a456-426614174999"
				username := "legacycamel"
				email := "legacycamel@example.com"

				mc := jwt.MapClaims{
					"userId":   userID,
					"username": username,
					"email":    email,
				}
				tok := jwt.NewWithClaims(jwt.SigningMethodHS256, mc)
				resigned, err := tok.SignedString([]byte("test-secret-key-for-middleware-testing"))
				Expect(err).ToNot(HaveOccurred())

				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Bearer "+resigned)
				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(BeTrue())
				Expect(capturedUser).ToNot(BeNil())
				Expect(capturedUser.UserID).To(Equal(userID))
				Expect(capturedUser.Username).To(Equal(username))
				Expect(capturedUser.Email).To(Equal(email))
			})
		})
	})

	Describe("User Context Helpers", func() {
		Context("when extracting user from context", func() {
			It("should return user context when present", func() {
				userCtx := &UserContext{
					UserID:   "user123",
					Username: "testuser",
					Email:    "test@example.com",
				}

				ctx := context.WithValue(context.Background(), UserContextKey, userCtx)

				extractedUser, ok := GetUserFromContext(ctx)
				Expect(ok).To(BeTrue())
				Expect(extractedUser).To(Equal(userCtx))
			})

			It("should return false when no user context is present", func() {
				ctx := context.Background()

				extractedUser, ok := GetUserFromContext(ctx)
				Expect(ok).To(BeFalse())
				Expect(extractedUser).To(BeNil())
			})

			It("should return false when context contains wrong type", func() {
				ctx := context.WithValue(context.Background(), UserContextKey, "not-a-user-context")

				extractedUser, ok := GetUserFromContext(ctx)
				Expect(ok).To(BeFalse())
				Expect(extractedUser).To(BeNil())
			})
		})
	})

	Describe("Edge Cases", func() {
		Context("when handling malformed requests", func() {
			It("should handle requests with very long authorization headers", func() {
				longToken := "Bearer " + string(make([]byte, 10000)) // Very long invalid token
				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", longToken)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
			})

			It("should handle authorization header with only 'Bearer'", func() {
				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Bearer")

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
			})

			It("should handle authorization header with extra spaces", func() {
				userID := "123e4567-e89b-12d3-a456-426614174000"
				username := "testuser"
				email := "test@example.com"

				token, err := jwtService.GenerateToken(userID, username, email)
				Expect(err).ToNot(HaveOccurred())

				req := httptest.NewRequest("POST", "/findMusicMatches", nil)
				req.Header.Set("Authorization", "Bearer  "+token) // Extra space

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				// Should still work because we trim the prefix
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(nextCalled).To(BeTrue())
				Expect(capturedUser.Username).To(Equal(username))
			})
		})

		Context("when handling different HTTP methods", func() {
			It("should protect GET requests", func() {
				req := httptest.NewRequest("GET", "/protectedResource", nil)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
			})

			It("should protect PUT requests", func() {
				req := httptest.NewRequest("PUT", "/protectedResource", nil)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
			})

			It("should protect DELETE requests", func() {
				req := httptest.NewRequest("DELETE", "/protectedResource", nil)

				middleware.Middleware(nextHandler).ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(nextCalled).To(BeFalse())
			})
		})
	})
})
