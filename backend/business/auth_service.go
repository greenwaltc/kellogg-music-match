package business

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"golang.org/x/crypto/bcrypt"
)

// AuthService implements the business logic for authentication
type AuthService struct {
	userRepo UserRepository
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
}

// RegisterUser implements user registration business logic
func (s *AuthService) RegisterUser(ctx context.Context, registerRequest generated.RegisterRequest) (generated.ImplResponse, error) {
	// Validate input
	if registerRequest.Username == "" || registerRequest.Password == "" ||
		registerRequest.Email == "" || registerRequest.FirstName == "" ||
		registerRequest.LastName == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "all fields are required",
		}), nil
	}

	// Check if user already exists
	userExists, err := s.userRepo.UserExistsByUsername(ctx, registerRequest.Username)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to check username availability",
		}), nil
	}
	if userExists {
		return generated.Response(http.StatusConflict, generated.ErrorResponse{
			Message: "username already exists",
		}), nil
	}

	emailExists, err := s.userRepo.UserExistsByEmail(ctx, registerRequest.Email)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to check email availability",
		}), nil
	}
	if emailExists {
		return generated.Response(http.StatusConflict, generated.ErrorResponse{
			Message: "email already exists",
		}), nil
	}

	// Generate unique ID
	userID := uuid.New()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registerRequest.Password), bcrypt.DefaultCost)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to process password",
		}), nil
	}

	// Create user
	dbUser, err := s.userRepo.CreateUser(
		ctx,
		userID,
		registerRequest.Username,
		registerRequest.Email,
		registerRequest.FirstName,
		registerRequest.LastName,
		string(hashedPassword),
	)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to create user",
		}), nil
	}

	// Convert database user to API user
	user := &generated.User{
		Id:        dbUser.ID.String(), // Standard UUID format with dashes
		Username:  dbUser.Username,
		Email:     dbUser.Email,
		FirstName: dbUser.FirstName,
		LastName:  dbUser.LastName,
		Artists:   []string{}, // Will be populated when user sets artists
	}

	// Return response
	response := generated.AuthResponse{
		User: *user,
	}
	return generated.Response(http.StatusCreated, response), nil
}

// LoginUser implements user authentication business logic
func (s *AuthService) LoginUser(ctx context.Context, loginRequest generated.LoginRequest) (generated.ImplResponse, error) {
	// Validate input
	username := loginRequest.Username
	password := loginRequest.Password

	if username == "" || password == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "username and password are required",
		}), nil
	}

	// Authenticate user
	dbUser, err := s.userRepo.GetUserByUsernameWithPassword(ctx, username)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "authentication failed",
		}), nil
	}

	if dbUser == nil {
		return generated.Response(http.StatusUnauthorized, generated.ErrorResponse{
			Message: "invalid username or password",
		}), nil
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(password))
	if err != nil {
		return generated.Response(http.StatusUnauthorized, generated.ErrorResponse{
			Message: "invalid username or password",
		}), nil
	}

	// Get user's artists
	artists, err := s.userRepo.GetUserArtists(ctx, dbUser.ID)
	if err != nil {
		// Log error but don't fail authentication
		artists = []sqlc.Artist{}
	}

	// Convert to string slice
	artistNames := make([]string, len(artists))
	for i, artist := range artists {
		artistNames[i] = artist.Name
	}

	// Convert database user to API user
	user := &generated.User{
		Id:        dbUser.ID.String(), // Standard UUID format with dashes
		Username:  dbUser.Username,
		Email:     dbUser.Email,
		FirstName: dbUser.FirstName,
		LastName:  dbUser.LastName,
		Artists:   artistNames,
	}

	// Return response
	response := generated.AuthResponse{
		User: *user,
	}
	return generated.Response(http.StatusOK, response), nil
}
