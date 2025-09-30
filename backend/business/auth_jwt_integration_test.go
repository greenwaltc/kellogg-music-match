package business_test

import (
	"context"
	"net/http"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auth Service JWT Integration", func() {
	var (
		authService *business.AuthService
		jwtService  *business.JWTService
		userRepo    *MockUserRepository
	)

	BeforeEach(func() {
		jwtConfig := &config.JWTConfig{
			SecretKey:    "test-secret-key-for-auth-integration",
			ExpiryHours:  24,
			RefreshHours: 168,
		}
		jwtService = business.NewJWTService(jwtConfig)
		userRepo = NewMockUserRepository()
		authService = business.NewAuthService(userRepo, jwtService)
	})

	Describe("User Login with JWT", func() {
		Context("when logging in with valid credentials", func() {
			It("should return a valid JWT token", func() {
				loginRequest := generated.LoginRequest{
					Username: "testuser",
					Password: "correctpassword",
				}

				// Configure mock for successful login
				userRepo.SetGetUserByUsernameWithPasswordResult(&MockUserWithPassword{
					MockUser: MockUser{
						ID:        "123e4567-e89b-12d3-a456-426614174000",
						Username:  "testuser",
						Email:     "test@example.com",
						FirstName: "Test",
						LastName:  "User",
					},
					PasswordHash: "$2a$10$xrw7VIXZ/766FZg/Xq6IpemPWXpoYhMd8xYg.5l14wO0AGfcCiyW2", // "correctpassword"
				}, nil)
				userRepo.SetGetUserArtistsResult([]string{"The Beatles", "Queen"}, nil)

				response, err := authService.LoginUser(context.Background(), loginRequest)

				Expect(err).ToNot(HaveOccurred())
				Expect(response.Code).To(Equal(http.StatusOK))

				// Extract the auth response
				authResponse, ok := response.Body.(generated.AuthResponse)
				Expect(ok).To(BeTrue())
				Expect(authResponse.Token).ToNot(BeNil())
				Expect(*authResponse.Token).ToNot(BeEmpty())

				// Validate the token
				claims, err := jwtService.ValidateToken(*authResponse.Token)
				Expect(err).ToNot(HaveOccurred())
				Expect(claims.UserID).To(Equal("123e4567-e89b-12d3-a456-426614174000"))
				Expect(claims.Username).To(Equal("testuser"))
				Expect(claims.Email).To(Equal("test@example.com"))
			})
		})
	})
})
