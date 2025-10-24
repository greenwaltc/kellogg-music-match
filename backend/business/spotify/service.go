package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/crypto"
	database "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
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
	UserID       uuid.UUID
}

// Service coordinates Spotify sync logic (stub implementation)
type Service struct {
	mu                sync.Mutex
	jobs              map[string]*SyncJob // keyed by username
	lastStart         map[string]time.Time
	store             TokenStore
	encKey            string
	httpClient        *http.Client
	clientID          string
	clientSecret      string
	redirectURI       string
	apiBase           string
	tokenURL          string
	tokenWaitInterval time.Duration
	tokenWaitTimeout  time.Duration

	// test hook overrides
	fetchOverride func(ctx context.Context, username, rng string) ([]business.SpotifyTopArtist, []business.SpotifyTopTrack, error)
}

var ErrSyncInProgress = errors.New("sync already in progress")
var ErrRateLimited = errors.New("sync recently started; please wait")

// Cooldown between successive sync starts
const cooldown = 5 * time.Second

// Cancelled status (extended)
const StatusCancelled = "cancelled"

// TokenStore defines persistence needed for Spotify tokens (subset of repository)
type TokenStore interface {
	UpsertSpotifyTokens(ctx context.Context, userID uuid.UUID, accessToken string, refreshTokenEncrypted []byte, expiresAt time.Time, scope string, tokenType string) error
	StoreSpotifyTopArtists(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []business.SpotifyTopArtist) error
	StoreSpotifyTopTracks(ctx context.Context, userID uuid.UUID, fetchedAt time.Time, rng string, items []business.SpotifyTopTrack) error
	GetSpotifyTokensByUser(ctx context.Context, userID uuid.UUID) (*database.SpotifyToken, error)
}

// Functional options
type Option func(*Service)

func WithHTTPClient(c *http.Client) Option {
	return func(s *Service) {
		if c != nil {
			s.httpClient = c
		}
	}
}
func WithSpotifyCredentials(clientID, clientSecret, redirectURI string) Option {
	return func(s *Service) {
		s.clientID = clientID
		s.clientSecret = clientSecret
		s.redirectURI = redirectURI
	}
}
func WithBaseURLs(apiBase, tokenURL string) Option {
	return func(s *Service) {
		if apiBase != "" {
			s.apiBase = apiBase
		}
		if tokenURL != "" {
			s.tokenURL = tokenURL
		}
	}
}
func WithTokenWait(interval, timeout time.Duration) Option {
	return func(s *Service) {
		if interval > 0 {
			s.tokenWaitInterval = interval
		}
		if timeout > 0 {
			s.tokenWaitTimeout = timeout
		}
	}
}

// NewService constructs a spotify Service. If store is nil, persistence is disabled (tokens stay in-memory only)
func NewService(store TokenStore, encryptionKey string, opts ...Option) *Service {
	s := &Service{
		jobs:              make(map[string]*SyncJob),
		lastStart:         make(map[string]time.Time),
		store:             store,
		encKey:            encryptionKey,
		httpClient:        &http.Client{Timeout: 15 * time.Second},
		apiBase:           "https://api.spotify.com/v1",
		tokenURL:          "https://accounts.spotify.com/api/token",
		tokenWaitInterval: 150 * time.Millisecond,
		tokenWaitTimeout:  3 * time.Second,
	}
	for _, o := range opts {
		o(s)
	}
	return s
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

// runJob transitions through states and performs (stub) ingestion of Spotify top items.
// Real implementation would call Spotify Web API; here we generate placeholder data and persist.
func (s *Service) runJob(username string) {
	// Wait for tokens
	waited := time.Duration(0)
	for {
		s.mu.Lock()
		job := s.jobs[username]
		if job == nil {
			s.mu.Unlock()
			return
		}
		if job.Status == StatusCancelled {
			s.mu.Unlock()
			return
		}
		if job.AccessToken != "" {
			job.Status = StatusInProgress
			job.Progress = 5
			s.mu.Unlock()
			break
		}
		s.mu.Unlock()
		if waited >= s.tokenWaitTimeout {
			s.mu.Lock()
			if job := s.jobs[username]; job != nil {
				job.Status = StatusFailed
				job.Message = "Timed out waiting for token exchange"
				finished := time.Now().UTC()
				job.FinishedAt = &finished
			}
			s.mu.Unlock()
			return
		}
		time.Sleep(s.tokenWaitInterval)
		waited += s.tokenWaitInterval
	}

	ranges := []string{"short_term", "medium_term", "long_term"}
	perRangeProgress := int32((100 - 5) / int32(len(ranges)))
	succeeded := []string{}
	failed := []string{}
	for i, rng := range ranges {
		var artists []business.SpotifyTopArtist
		var tracks []business.SpotifyTopTrack
		var err error
		if s.fetchOverride != nil {
			artists, tracks, err = s.fetchOverride(context.Background(), username, rng)
		} else {
			artists, tracks, err = s.fetchTopItems(context.Background(), username, rng)
		}
		if err != nil {
			logger.L().Error("spotify.sync.fetch.error", "user", username, "range", rng, "err", err)
			failed = append(failed, rng+":"+err.Error())
		} else {
			succeeded = append(succeeded, rng)
			if s.store != nil {
				s.mu.Lock()
				job := s.jobs[username]
				userID := uuid.Nil
				if job != nil {
					userID = job.UserID
				}
				s.mu.Unlock()
				if userID != uuid.Nil {
					fetchedAt := time.Now().UTC()
					if err := s.store.StoreSpotifyTopArtists(context.Background(), userID, fetchedAt, rng, artists); err != nil {
						logger.L().Error("spotify.sync.persist.artists.error", "err", err)
					}
					if err := s.store.StoreSpotifyTopTracks(context.Background(), userID, fetchedAt, rng, tracks); err != nil {
						logger.L().Error("spotify.sync.persist.tracks.error", "err", err)
					}
				} else {
					logger.L().Warn("spotify.sync.persist.skipped.missing_user_id", "user", username, "range", rng)
				}
			}
		}
		s.mu.Lock()
		if job := s.jobs[username]; job != nil && job.Status == StatusInProgress {
			job.Progress = 5 + perRangeProgress*int32(i+1)
		}
		s.mu.Unlock()
	}
	// Finalize status
	s.mu.Lock()
	if job := s.jobs[username]; job != nil && (job.Status == StatusInProgress) {
		finished := time.Now().UTC()
		job.FinishedAt = &finished
		if len(succeeded) == 0 {
			job.Status = StatusFailed
			job.Message = "All ranges failed"
		} else {
			job.Status = StatusComplete
			job.Progress = 100
			if len(failed) > 0 {
				job.Message = fmt.Sprintf("Partial success. Succeeded: %v Failed: %v", succeeded, failed)
			} else {
				job.Message = "Sync finished"
			}
		}
		logger.L().Info("spotify.sync.complete", "user", username, "succeeded", succeeded, "failed", failed)
	}
	s.mu.Unlock()
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

// ExchangeCodeForTokens performs the authorization code exchange against Spotify Accounts service.
// Supports optional PKCE by supplying a non-empty codeVerifier.
func (s *Service) ExchangeCodeForTokens(ctx context.Context, code string, codeVerifier string) (string, string, int, error) {
	if s.clientID == "" || s.clientSecret == "" || s.redirectURI == "" {
		return "", "", 0, errors.New("spotify credentials not configured")
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", s.redirectURI)
	if codeVerifier != "" { // PKCE
		form.Set("code_verifier", codeVerifier)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.clientID, s.clientSecret)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		snippet := string(body)
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		// Log detailed error server-side but return generic to caller.
		logger.L().Error("spotify.token.exchange.failed", "status", resp.StatusCode, "body", snippet)
		return "", "", 0, fmt.Errorf("token exchange failed status=%d", resp.StatusCode)
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", 0, err
	}
	return parsed.AccessToken, parsed.RefreshToken, parsed.ExpiresIn, nil
}

// ExchangeCodeForTokensWithRedirect performs the authorization code exchange using an explicit redirect URI.
// This supports mobile clients that use a custom scheme (e.g., affyne://spotify/callback) while allowing the
// server default redirect to remain configured for web flows.
func (s *Service) ExchangeCodeForTokensWithRedirect(ctx context.Context, code string, codeVerifier string, redirectURI string) (string, string, int, error) {
	if s.clientID == "" || s.clientSecret == "" || redirectURI == "" {
		return "", "", 0, errors.New("spotify credentials/redirect not configured")
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	if codeVerifier != "" { // PKCE
		form.Set("code_verifier", codeVerifier)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.clientID, s.clientSecret)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		snippet := string(body)
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		logger.L().Error("spotify.token.exchange.failed", "status", resp.StatusCode, "body", snippet)
		return "", "", 0, fmt.Errorf("token exchange failed status=%d", resp.StatusCode)
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", 0, err
	}
	return parsed.AccessToken, parsed.RefreshToken, parsed.ExpiresIn, nil
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

// RefreshAccessToken exchanges a refresh_token for a new access token (and possibly a new refresh token).
func (s *Service) RefreshAccessToken(ctx context.Context, refreshToken string) (string, string, int, error) {
	if s.clientID == "" || s.clientSecret == "" {
		return "", "", 0, errors.New("spotify credentials not configured")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.clientID, s.clientSecret)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		snippet := string(body)
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		logger.L().Error("spotify.token.refresh.failed", "status", resp.StatusCode, "body", snippet)
		return "", "", 0, fmt.Errorf("token refresh failed status=%d", resp.StatusCode)
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", 0, err
	}
	return parsed.AccessToken, parsed.RefreshToken, parsed.ExpiresIn, nil
}

// RefreshUsingStoredTokens starts a sync using the stored refresh token to obtain a new access token.
func (s *Service) RefreshUsingStoredTokens(ctx context.Context, username string, userID uuid.UUID) (*SyncJob, error) {
	// Prevent starting if in-progress
	s.mu.Lock()
	if existing, ok := s.jobs[username]; ok {
		if existing.Status == StatusPending || existing.Status == StatusInProgress {
			s.mu.Unlock()
			return nil, ErrSyncInProgress
		}
	}
	s.mu.Unlock()

	if s.store == nil {
		return nil, errors.New("token store not configured")
	}
	tok, err := s.store.GetSpotifyTokensByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if tok == nil || len(tok.RefreshTokenEncrypted) == 0 {
		return nil, errors.New("no stored tokens")
	}
	if s.encKey == "" {
		return nil, errors.New("encryption key not configured")
	}
	// Decrypt stored refresh token
	rtBytes, err := crypto.DecryptAESGCM(tok.RefreshTokenEncrypted, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}
	refreshToken := string(rtBytes)
	accessToken, newRefresh, expiresIn, err := s.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	// Start a new job and inject tokens
	job := s.StartSync(ctx, username, "", "")
	// If StartSync rate-limited us, report as error
	if job.Status == StatusFailed && strings.Contains(strings.ToLower(job.Message), "rate") {
		return nil, ErrRateLimited
	}
	if newRefresh == "" {
		newRefresh = refreshToken
	}
	s.SetJobTokens(username, userID, accessToken, newRefresh, expiresIn)
	// Best effort persist
	_ = s.PersistTokens(ctx, userID, accessToken, newRefresh, expiresIn, "")
	return job, nil
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

// SetJobTokens populates an existing job (created by StartSync) with token info and userID.
// Called by the HTTP wrapper after successful token exchange & persistence.
func (s *Service) SetJobTokens(username string, userID uuid.UUID, accessToken, refreshToken string, expiresIn int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[username]
	if !ok {
		return
	}
	job.AccessToken = accessToken
	job.RefreshToken = refreshToken
	if expiresIn > 0 {
		exp := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
		job.ExpiresAt = &exp
	}
	job.UserID = userID
}

// ----- Spotify API fetch helpers -----
type spotifyImage struct {
	URL    string `json:"url"`
	Width  *int   `json:"width"`
	Height *int   `json:"height"`
}

type spotifyArtist struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Genres     []string       `json:"genres"`
	Popularity *int           `json:"popularity"`
	Images     []spotifyImage `json:"images"`
}
type spotifyTrack struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	DurationMS *int            `json:"duration_ms"`
	Popularity *int            `json:"popularity"`
	Artists    []spotifyArtist `json:"artists"`
	Album      struct {
		ID     string         `json:"id"`
		Name   string         `json:"name"`
		Images []spotifyImage `json:"images"`
	} `json:"album"`
}

// fetchTopItems gets top artists & tracks for a single time range
func (s *Service) fetchTopItems(ctx context.Context, username string, rng string) ([]business.SpotifyTopArtist, []business.SpotifyTopTrack, error) {
	s.mu.Lock()
	job := s.jobs[username]
	token := ""
	if job != nil {
		token = job.AccessToken
	}
	s.mu.Unlock()
	if token == "" {
		return nil, nil, errors.New("missing access token")
	}
	artistsResp, err := s.fetchAll(ctx, token, fmt.Sprintf("%s/me/top/artists?time_range=%s&limit=50", s.apiBase, rng))
	if err != nil {
		return nil, nil, err
	}
	tracksResp, err := s.fetchAll(ctx, token, fmt.Sprintf("%s/me/top/tracks?time_range=%s&limit=50", s.apiBase, rng))
	if err != nil {
		return nil, nil, err
	}
	artists := []business.SpotifyTopArtist{}
	for i, a := range artistsResp.Artists {
		var pop *int32
		if a.Popularity != nil {
			p := int32(*a.Popularity)
			pop = &p
		}
		var img *string
		if u := pickImageURL(a.Images); u != "" {
			img = &u
		}
		artists = append(artists, business.SpotifyTopArtist{
			Rank:            int32(i + 1),
			SpotifyArtistID: a.ID,
			Name:            a.Name,
			Genres:          a.Genres,
			Popularity:      pop,
			ImageURL:        img,
		})
	}
	tracks := []business.SpotifyTopTrack{}
	for i, t := range tracksResp.Tracks {
		var pop *int32
		if t.Popularity != nil {
			p := int32(*t.Popularity)
			pop = &p
		}
		var dur *int32
		if t.DurationMS != nil {
			d := int32(*t.DurationMS)
			dur = &d
		}
		artistNames := []string{}
		artistIDs := []string{}
		for _, ar := range t.Artists {
			artistNames = append(artistNames, ar.Name)
			artistIDs = append(artistIDs, ar.ID)
		}
		var img *string
		if u := pickImageURL(t.Album.Images); u != "" {
			img = &u
		}
		var albumName *string
		if strings.TrimSpace(t.Album.Name) != "" {
			n := t.Album.Name
			albumName = &n
		}
		var albumID *string
		if strings.TrimSpace(t.Album.ID) != "" {
			id := t.Album.ID
			albumID = &id
		}
		tracks = append(tracks, business.SpotifyTopTrack{
			Rank:           int32(i + 1),
			SpotifyTrackID: t.ID,
			Name:           t.Name,
			ArtistNames:    artistNames,
			ArtistIDs:      artistIDs,
			Popularity:     pop,
			DurationMS:     dur,
			ImageURL:       img,
			AlbumName:      albumName,
			AlbumID:        albumID,
		})
	}
	return artists, tracks, nil
}

// unified paging (Spotify returns next URL if more pages)
type topArtistsPayload struct {
	Items []spotifyArtist `json:"items"`
	Next  *string         `json:"next"`
}
type topTracksPayload struct {
	Items []spotifyTrack `json:"items"`
	Next  *string        `json:"next"`
}

func (s *Service) fetchAll(ctx context.Context, token, firstURL string) (struct {
	Artists []spotifyArtist
	Tracks  []spotifyTrack
}, error) {
	result := struct {
		Artists []spotifyArtist
		Tracks  []spotifyTrack
	}{}
	// Determine entity from URL path
	isArtists := strings.Contains(firstURL, "/artists")
	nextURL := firstURL
	for nextURL != "" {
		attempts := 0
		for {
			attempts++
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
			if err != nil {
				return result, err
			}
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := s.httpClient.Do(req)
			if err != nil {
				return result, err
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 429 { // rate limited
				retryAfter := time.Duration(0)
				if ra := resp.Header.Get("Retry-After"); ra != "" {
					if secs, err := strconv.Atoi(ra); err == nil {
						retryAfter = time.Duration(secs) * time.Second
					}
				}
				if retryAfter == 0 {
					retryAfter = time.Duration(attempts) * 500 * time.Millisecond
				}
				if attempts >= 3 {
					return result, fmt.Errorf("spotify api rate limited after retries status=429 body=%s", string(body))
				}
				time.Sleep(retryAfter)
				continue
			}
			if resp.StatusCode != 200 {
				return result, fmt.Errorf("spotify api error status=%d body=%s", resp.StatusCode, string(body))
			}
			if isArtists {
				var p topArtistsPayload
				if err := json.Unmarshal(body, &p); err != nil {
					return result, err
				}
				result.Artists = append(result.Artists, p.Items...)
				if p.Next != nil {
					nextURL = *p.Next
				} else {
					nextURL = ""
				}
			} else {
				var p topTracksPayload
				if err := json.Unmarshal(body, &p); err != nil {
					return result, err
				}
				result.Tracks = append(result.Tracks, p.Items...)
				if p.Next != nil {
					nextURL = *p.Next
				} else {
					nextURL = ""
				}
			}
			break
		}
		// Safety cap (in case of loop) - should not exceed 5 pages
		if len(result.Artists)+len(result.Tracks) > 250 {
			break
		}
	}
	return result, nil
}

// pickImageURL chooses a reasonable image URL from a list of Spotify images.
// Preference order: medium (200-400px) then largest then smallest.
func pickImageURL(images []spotifyImage) string {
	if len(images) == 0 {
		return ""
	}
	// Prefer medium size
	bestURL := ""
	for _, im := range images {
		if im.Width != nil {
			w := *im.Width
			if w >= 200 && w <= 400 {
				return im.URL
			}
		}
	}
	// Fallback: largest
	maxW := -1
	for _, im := range images {
		if im.Width != nil && *im.Width > maxW {
			maxW = *im.Width
			bestURL = im.URL
		}
	}
	if bestURL != "" {
		return bestURL
	}
	// Final: first
	return images[0].URL
}
