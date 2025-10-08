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

// This test ensures that similarity scores returned by FindMusicMatches are sorted in non-increasing order
// (highest score first) when multiple overlapping users exist.
var _ = Describe("Spotify Similarity Ordering", func() {
	var (
		repo business.UserRepository
		svc  *business.MatchingService
		ctx  context.Context
	)

	makeUser := func(base string, artistIDs []string) string {
		id := uuid.New()
		username := base + "_" + id.String()[:8]
		pw := "$2a$10$abcdefghijklmnopqrstuv"
		_, err := repo.CreateUser(ctx, id, username, username+"@test.com", "Order", "User", pw, "2Y", 2026)
		Expect(err).NotTo(HaveOccurred())
		items := make([]business.SpotifyTopArtist, 0, len(artistIDs))
		for i, aid := range artistIDs {
			items = append(items, business.SpotifyTopArtist{Rank: int32(i + 1), SpotifyArtistID: aid, Name: aid})
		}
		pr := repo.(*business.PostgreSQLUserRepository)
		err = pr.StoreSpotifyTopArtists(ctx, id, time.Now(), "medium_term", items)
		Expect(err).NotTo(HaveOccurred())
		return username
	}

	BeforeEach(func() {
		var err error
		repo, err = business.NewUserRepository()
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
		svc = business.NewMatchingService(repo, business.NewMatchingEngine())
	})

	It("returns scores in non-increasing order", func() {
		// Anchor user
		anchor := makeUser("anchor_ord", []string{"a1", "a2", "a3", "a4"})
		// Various overlaps with different rank combinations
		_ = makeUser("high_overlap", []string{"a1", "a2", "a3", "a4"}) // should be high
		_ = makeUser("mid_overlap", []string{"a1", "a2", "x1", "x2"})  // medium
		_ = makeUser("low_overlap", []string{"a4", "x3", "x4", "x5"})  // lower
		_ = makeUser("tiny_overlap", []string{"a4"})                   // tiny
		_ = makeUser("no_overlap", []string{"z1", "z2"})               // will not contribute

		resp, err := svc.FindMusicMatches(ctx, generated.ArtistsRequest{Artists: []string{"ignored"}}, anchor, "medium_term", 10, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Code).To(Equal(200))
		list, ok := resp.Body.([]*generated.MatchUser)
		Expect(ok).To(BeTrue())
		// Filter out fallback/self placeholder if present
		filtered := make([]*generated.MatchUser, 0, len(list))
		for _, m := range list {
			if len(m.Artists) > 0 && m.Artists[0] == "(Need more Spotify data to compute real matches)" {
				continue
			}
			filtered = append(filtered, m)
		}
		for i := 1; i < len(filtered); i++ {
			Expect(filtered[i-1].Score).To(BeNumerically(">=", filtered[i].Score), "scores not sorted at position %d", i)
		}
	})
})
