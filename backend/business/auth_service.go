package business

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// AuthService implements the business logic for authentication
type AuthService struct {
	store *Store
}

// NewAuthService creates a new authentication service
func NewAuthService(store *Store) *AuthService {
	return &AuthService{
		store: store,
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
	if s.store.UserExists(registerRequest.Username) {
		return generated.Response(http.StatusConflict, generated.ErrorResponse{
			Message: "username already exists",
		}), nil
	}

	if s.store.EmailExists(registerRequest.Email) {
		return generated.Response(http.StatusConflict, generated.ErrorResponse{
			Message: "email already exists", 
		}), nil
	}

	// Generate unique ID
	idBytes := make([]byte, 16)
	rand.Read(idBytes)
	userID := hex.EncodeToString(idBytes)

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registerRequest.Password), bcrypt.DefaultCost)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to process password",
		}), nil
	}

	// Create user
	user := &generated.User{
		Id:        userID,
		Username:  registerRequest.Username,
		Email:     registerRequest.Email,
		FirstName: registerRequest.FirstName,
		LastName:  registerRequest.LastName,
		Artists:   []string{},
	}

	err = s.store.CreateUser(user, string(hashedPassword))
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: "failed to create user",
		}), nil
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
	user, err := s.store.AuthenticateUser(username, password)
	if err != nil {
		return generated.Response(http.StatusUnauthorized, generated.ErrorResponse{
			Message: "invalid username or password",
		}), nil
	}

	// Return response
	response := generated.AuthResponse{
		User: *user,
	}
	return generated.Response(http.StatusOK, response), nil
}