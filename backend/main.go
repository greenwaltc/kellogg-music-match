package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type User struct {
	Email    string   `json:"email"`
	FullName string   `json:"fullName"`
	Artists  []string `json:"artists,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	FullName string `json:"fullName"`
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
	users map[string]*User // key=email
}

var store = memoryStore{users: make(map[string]*User)}

func (s *memoryStore) upsertUser(email, fullName string) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[email]
	if !ok {
		u = &User{Email: email, FullName: fullName}
		s.users[email] = u
	} else {
		// update full name if changed
		u.FullName = fullName
	}
	return u
}

func (s *memoryStore) updateArtists(email string, artists []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[email]; ok {
		u.Artists = artists
	}
}

func (s *memoryStore) snapshotUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, &User{Email: u.Email, FullName: u.FullName, Artists: append([]string(nil), u.Artists...)})
	}
	return out
}

func main() {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200", "http://127.0.0.1:4200", "*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-User-Email"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
	})

	r.Post("/login", handleLogin)
	r.Post("/findMusicMatches", handleFindMatches)

	addr := ":8080"
	log.Printf("backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.FullName == "" || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "email and fullName required")
		return
	}
	u := store.upsertUser(req.Email, req.FullName)
	writeJSON(w, http.StatusOK, map[string]any{"email": u.Email, "fullName": u.FullName, "artists": u.Artists})
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
	// For demo we piggy-back on an X-User-Email header OR first existing user with no artists
	caller := strings.TrimSpace(strings.ToLower(r.Header.Get("X-User-Email")))
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
		if caller != "" && u.Email == caller { // skip self if we know caller
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
		res = append(res, MatchUser{Name: u.FullName, Overlap: ov, Score: math.Round(score*1000) / 1000})
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
