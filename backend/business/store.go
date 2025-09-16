package business

import (
	"errors"
	"sync"

	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"golang.org/x/crypto/bcrypt"
)

// UserWithPassword extends User with password hash for internal storage
// Deprecated: This type is no longer used. Use UserRepository interface instead.
type UserWithPassword struct {
	generated.User
	PasswordHash string `json:"-"`
}

// Store represents an in-memory data store for users
// Deprecated: This implementation has been replaced by UserRepository interface with PostgreSQL backend.
// Use business.NewUserRepository() instead of business.NewStore().
type Store struct {
	mu     sync.RWMutex
	users  map[string]*UserWithPassword // key=username
	emails map[string]string            // key=email, value=username
}

// NewStore creates a new memory store
// Deprecated: This implementation has been replaced by UserRepository interface with PostgreSQL backend.
// Use business.NewUserRepository() instead.
func NewStore() *Store {
	return &Store{
		users:  make(map[string]*UserWithPassword),
		emails: make(map[string]string),
	}
}

// UserExists checks if a username already exists
func (s *Store) UserExists(username string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.users[username]
	return exists
}

// EmailExists checks if an email already exists
func (s *Store) EmailExists(email string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.emails[email]
	return exists
}

// CreateUser creates a new user with the given password hash
func (s *Store) CreateUser(user *generated.User, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check for race conditions
	if _, exists := s.users[user.Username]; exists {
		return errors.New("username already exists")
	}
	if _, exists := s.emails[user.Email]; exists {
		return errors.New("email already exists")
	}

	userWithPassword := &UserWithPassword{
		User:         *user,
		PasswordHash: passwordHash,
	}

	s.users[user.Username] = userWithPassword
	s.emails[user.Email] = user.Username
	return nil
}

// AuthenticateUser validates credentials and returns the user if valid
func (s *Store) AuthenticateUser(username, password string) (*generated.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, errors.New("user not found")
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	return &user.User, nil
}

// SnapshotUsers returns a snapshot of all users for matching
func (s *Store) SnapshotUsers() []*generated.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*generated.User, 0, len(s.users))
	for _, userWithPassword := range s.users {
		userCopy := userWithPassword.User
		users = append(users, &userCopy)
	}
	return users
}
