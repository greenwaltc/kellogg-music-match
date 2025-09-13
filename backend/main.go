package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string   `json:"id"`
	Username     string   `json:"username"`
	Email        string   `json:"email"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	PasswordHash string   `json:"-"` // Never return password hash in JSON
	Artists      []string `json:"artists,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

type AuthResponse struct {
	User  User   `json:"user"`
	Token string `json:"token,omitempty"`
}

type ArtistsRequest struct {
	Artists []string `json:"artists"`
}

type MatchUser struct {
	Name    string  `json:"name"`
	Overlap int     `json:"overlap"`
	Score   float64 `json:"score"`
}

type memoryStore struct {
	mu    sync.RWMutex
	users map[string]*User // key=username
}

var store = memoryStore{users: make(map[string]*User)}

// Password hashing utilities
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Generate a simple user ID
func generateUserID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *memoryStore) createUser(username, email, firstName, lastName, password string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if username already exists
	if _, exists := s.users[username]; exists {
		return nil, fmt.Errorf("username already exists")
	}

	// Check if email already exists
	for _, u := range s.users {
		if u.Email == email {
			return nil, fmt.Errorf("email already exists")
		}
	}

	// Hash password
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password")
	}

	// Create user
	user := &User{
		ID:           generateUserID(),
		Username:     username,
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: hashedPassword,
		Artists:      []string{},
	}

	s.users[username] = user
	return user, nil
}

func (s *memoryStore) authenticateUser(username, password string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	if !checkPassword(password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

func (s *memoryStore) getUserByUsername(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

func (s *memoryStore) updateArtists(username string, artists []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[username]; ok {
		u.Artists = artists
	}
}

func (s *memoryStore) snapshotUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, &User{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Artists:   append([]string(nil), u.Artists...),
		})
	}
	return out
}

func main() {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200", "http://127.0.0.1:4200", "*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-User-Username"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
	})

	r.Post("/login", handleLogin)
	r.Post("/register", handleRegister)
	r.Post("/findMusicMatches", handleFindMatches)

	addr := ":8080"
	log.Printf("backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validate input
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if req.Username == "" || req.Email == "" || req.FirstName == "" || req.LastName == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username, email, firstName, lastName, and password are required")
		return
	}

	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	// Create user
	user, err := store.createUser(req.Username, req.Email, req.FirstName, req.LastName, req.Password)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	// Return response
	response := AuthResponse{
		User: *user,
	}
	writeJSON(w, http.StatusCreated, response)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validate input
	req.Username = strings.TrimSpace(req.Username)

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	// Authenticate user
	user, err := store.authenticateUser(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	// Return response
	response := AuthResponse{
		User: *user,
	}
	writeJSON(w, http.StatusOK, response)
}

func handleFindMatches(w http.ResponseWriter, r *http.Request) {
	var req ArtistsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Artists) == 0 {
		writeError(w, http.StatusBadRequest, "artists required")
		return
	}
	clean := dedupeAndNormalize(req.Artists)
	if len(clean) == 0 {
		writeError(w, http.StatusBadRequest, "no valid artists provided")
		return
	}
	// For demo we piggy-back on an X-User-Username header OR first existing user with no artists
	caller := strings.TrimSpace(r.Header.Get("X-User-Username"))
	if caller == "" {
		// No auth: treat as anonymous ephemeral user (not stored) unless already logged in previously via /login.
	}
	// If caller provided and exists, update their artist list
	if caller != "" {
		store.updateArtists(caller, clean)
	}
	all := store.snapshotUsers()
	matches := computeMatches(clean, caller, all)
	// Return top 5 only
	if len(matches) > 5 {
		matches = matches[:5]
	}
	writeJSON(w, http.StatusOK, matches)
}

func dedupeAndNormalize(in []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(in))
	for _, a := range in {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		al := strings.ToLower(a)
		if _, ok := seen[al]; ok {
			continue
		}
		seen[al] = struct{}{}
		out = append(out, a)
	}
	return out
}

func computeMatches(target []string, caller string, users []*User) []MatchUser {
	// Build set for overlap scoring
	set := make(map[string]struct{}, len(target))
	for _, a := range target {
		set[strings.ToLower(a)] = struct{}{}
	}
	res := make([]MatchUser, 0)
	for _, u := range users {
		if caller != "" && u.Username == caller { // skip self if we know caller
			continue
		}
		if len(u.Artists) == 0 { // skip users without preferences
			continue
		}
		ov := 0
		for _, a := range u.Artists {
			if _, ok := set[strings.ToLower(a)]; ok {
				ov++
			}
		}
		if ov == 0 {
			continue
		}
		// Score: Jaccard-like with emphasis on overlap count
		union := float64(len(set) + countUniqueLower(u.Artists) - ov)
		jaccard := float64(ov) / union
		// Weighted score: overlap weight 0.7 + jaccard 0.3
		score := 0.7*float64(ov)/float64(len(set)) + 0.3*jaccard
		res = append(res, MatchUser{Name: u.FirstName + " " + u.LastName, Overlap: ov, Score: math.Round(score*1000) / 1000})
	}
	// sort by score desc, then overlap desc, then name
	sort.Slice(res, func(i, j int) bool {
		if res[i].Score == res[j].Score {
			if res[i].Overlap == res[j].Overlap {
				return res[i].Name < res[j].Name
			}
			return res[i].Overlap > res[j].Overlap
		}
		return res[i].Score > res[j].Score
	})
	return res
}

func countUniqueLower(list []string) int {
	m := make(map[string]struct{})
	for _, a := range list {
		m[strings.ToLower(strings.TrimSpace(a))] = struct{}{}
	}
	return len(m)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"message": msg})
}
