package spotify

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business/crypto"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
)

// Status enumerates sync job states
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusComplete   = "complete"
	StatusFailed     = "failed"
)

// SyncJob holds state for a user's current/last sync
type SyncJob struct {
	Status       string
	Progress     int32
	StartedAt    *time.Time
	FinishedAt   *time.Time
	Message      string
	Code         string // captured authorization code (would be exchanged for tokens)
	State        string // original state value
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
}

// Service coordinates Spotify sync logic (stub implementation)
type Service struct {
	mu   sync.Mutex
	jobs map[string]*SyncJob // keyed by username
	// map of username -> time last started; used for simple rate limiting
	lastStart map[string]time.Time
	store     TokenStore
	encKey    string
}

var ErrSyncInProgress = errors.New("sync already in progress")
var ErrRateLimited = errors.New("sync recently started; please wait")

// Cooldown between successive sync starts
const cooldown = 5 * time.Second

// Cancelled status (extended)
const StatusCancelled = "cancelled"

// Minimum seconds after finish to allow restart without hitting cooldown
const postFinishGrace = 0 * time.Second

// TokenStore defines persistence needed for Spotify tokens (subset of repository)
type TokenStore interface {
	UpsertSpotifyTokens(ctx context.Context, userID uuid.UUID, accessToken string, refreshTokenEncrypted []byte, expiresAt time.Time, scope string, tokenType string) error
}

// NewService constructs a spotify Service. If store is nil, persistence is disabled (tokens stay in-memory only)
func NewService(store TokenStore, encryptionKey string) *Service {
	return &Service{jobs: make(map[string]*SyncJob), lastStart: make(map[string]time.Time), store: store, encKey: encryptionKey}
}

// StartSync registers a new sync job (or restarts if previous finished)
func (s *Service) StartSync(ctx context.Context, username, code, state string) *SyncJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.jobs[username]; ok {
		if existing.Status == StatusPending || existing.Status == StatusInProgress {
			return existing // already running
		}
	}
	if last, ok := s.lastStart[username]; ok {
		if time.Since(last) < cooldown {
			return &SyncJob{Status: StatusFailed, Progress: 0, Message: ErrRateLimited.Error()}
		}
	}
	// Initialize job
	now := time.Now().UTC()
	job := &SyncJob{
		Status:    StatusPending,
		Progress:  0,
		StartedAt: &now,
		Code:      code,
		State:     state,
		Message:   "Sync accepted",
	}
	s.jobs[username] = job
	s.lastStart[username] = now
	// Launch background progression
	go s.runJob(username)
	return job
}

// GetStatus returns a snapshot of the user's job state
func (s *Service) GetStatus(username string) *SyncJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[username]; ok {
		// Return a shallow copy to avoid external mutation
		copy := *job
		return &copy
	}
	return &SyncJob{Status: StatusComplete, Progress: 100, Message: "No active job"}
}

func (s *Service) runJob(username string) {
	stages := []int32{5, 25, 55, 80, 100}
	interval := 800 * time.Millisecond
	for i, p := range stages {
		time.Sleep(interval)
		s.mu.Lock()
		job := s.jobs[username]
		if job == nil {
			s.mu.Unlock()
			return
		}
		if job.Status == StatusCancelled { // stop progression
			logger.L().Info("spotify.sync.cancelled", "user", username)
			s.mu.Unlock()
			return
		}
		if i == 0 {
			job.Status = StatusInProgress
		}
		job.Progress = p
		if p == 100 {
			job.Status = StatusComplete
			finished := time.Now().UTC()
			job.FinishedAt = &finished
			job.Message = "Sync finished"
		}
		logger.L().Info("spotify.sync.progress", "user", username, "progress", p, "status", job.Status)
		s.mu.Unlock()
	}
}

// RetrySync allows restarting a sync only if previous job is in a terminal state (failed, cancelled, complete)
func (s *Service) RetrySync(ctx context.Context, username string, code, state string) (*SyncJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.jobs[username]; ok {
		switch existing.Status {
		case StatusPending, StatusInProgress:
			return nil, ErrSyncInProgress
		case StatusFailed, StatusCancelled, StatusComplete:
			// allow restart immediately ignoring cooldown
		}
	}
	now := time.Now().UTC()
	job := &SyncJob{Status: StatusPending, Progress: 0, StartedAt: &now, Code: code, State: state, Message: "Retry accepted"}
	s.jobs[username] = job
	s.lastStart[username] = now
	go s.runJob(username)
	return job, nil
}

// ExchangeCodeForTokens is a stub representing the server-side code->token exchange.
// In a real implementation this would call Spotify's /api/token endpoint with client secret.
func (s *Service) ExchangeCodeForTokens(ctx context.Context, code string) (accessToken string, refreshToken string, expiresIn int, err error) {
	// Stub: return synthetic values
	return "stub_access_token", "stub_refresh_token", 3600, nil
}

// PersistTokens encrypts and stores tokens if a store and encryption key are configured.
func (s *Service) PersistTokens(ctx context.Context, userID uuid.UUID, accessToken, refreshToken string, expiresIn int, scope string) error {
	if s.store == nil || s.encKey == "" {
		logger.L().Warn("spotify.tokens.persistence.disabled", "reason", "missing store or encryption key")
		return nil
	}
	ciphertext, err := crypto.EncryptAESGCM([]byte(refreshToken), s.encKey)
	if err != nil {
		return err
	}
	expiresAt := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
	if err := s.store.UpsertSpotifyTokens(ctx, userID, accessToken, ciphertext, expiresAt, scope, "Bearer"); err != nil {
		return err
	}
	logger.L().Info("spotify.tokens.persisted", "user_id", userID.String())
	return nil
}

// CancelSync transitions a running job to cancelled.
func (s *Service) CancelSync(username string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[username]
	if !ok {
		return false
	}
	if job.Status == StatusPending || job.Status == StatusInProgress {
		job.Status = StatusCancelled
		finished := time.Now().UTC()
		job.FinishedAt = &finished
		job.Message = "Sync cancelled"
		logger.L().Info("spotify.sync.cancel", "user", username)
		return true
	}
	return false
}
