package business_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

var _ = Describe("Spotify Similarity via MatchingService", func() {
	var (
		repo business.UserRepository
		svc  *business.MatchingService
		ctx  context.Context
	)

	beforeUser := func(username string, artistIDsAndNames []struct{ ID, Name string }) (uuid.UUID, string) {
		id := uuid.New()
		// Always append suffix to guarantee uniqueness across entire test process
		username = username + "_" + id.String()[:8]
		pw := "$2a$10$abcdefghijklmnopqrstuv" // dummy bcrypt (won't be verified)
		_, err := repo.CreateUser(ctx, id, username, username+"@test.com", "Test", "User", pw, "2Y", 2026)
		Expect(err).NotTo(HaveOccurred())
		// Build snapshot items
		items := make([]business.SpotifyTopArtist, 0, len(artistIDsAndNames))
		for i, a := range artistIDsAndNames {
			r := int32(i + 1)
			name := a.Name
			items = append(items, business.SpotifyTopArtist{Rank: r, SpotifyArtistID: a.ID, Name: name})
		}
		// Store snapshot (medium_term)
		psqlRepo := repo.(*business.PostgreSQLUserRepository)
		err = psqlRepo.StoreSpotifyTopArtists(ctx, id, time.Now(), "medium_term", items)
		Expect(err).NotTo(HaveOccurred())
		return id, username
	}

	BeforeEach(func() {
		var err error
		repo, err = business.NewUserRepository()
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
		engine := business.NewMatchingEngine()
		svc = business.NewMatchingService(repo, engine)
	})

	It("returns at least one similar user with overlap > 0 when overlapping snapshots exist", func() {
		// Anchor user
		anchorID, anchorUsername := beforeUser("anchor_user", []struct{ ID, Name string }{
			{"art1", "Artist One"},
			{"art2", "Artist Two"},
			{"art3", "Artist Three"},
		})
		// Similar user with two overlaps
		_, _ = beforeUser("similar_user", []struct{ ID, Name string }{
			{"art2", "Artist Two"},
			{"art3", "Artist Three"},
			{"artX", "Other"},
		})
		// Distant user with no overlap
		_, _ = beforeUser("distant_user", []struct{ ID, Name string }{
			{"artZ", "Zed"},
		})

		// Invoke matching using anchor username (artistsRequest ignored now)
		resp, err := svc.FindMusicMatches(ctx, generated.ArtistsRequest{Artists: []string{"placeholder"}}, anchorUsername, "medium_term", true, 10, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Code).To(Equal(200))
		list, ok := resp.Body.([]*generated.MatchUser)
		Expect(ok).To(BeTrue())
		// Log matches for debugging
		for _, m := range list {
			GinkgoWriter.Printf("MATCH: name=%s overlap=%d score=%f artists=%v\n", m.Name, m.Overlap, m.Score, m.Artists)
		}
		// Expect at least one overlap >=1 besides potential fallback (which has explanatory artist text)
		var found int
		for _, m := range list {
			if m.Overlap >= 1 && len(m.Artists) > 0 && m.Artists[0] != "(Need more Spotify data to compute real matches)" {
				found++
			}
		}
		Expect(found).To(BeNumerically(">=", 1))
		_ = anchorID // silence linter if unused
	})
})
