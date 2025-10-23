package business_test

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

var _ = Describe("Overlaps Limit Behavior", func() {
	var (
		repo business.UserRepository
		svc  *business.MatchingService
		ctx  context.Context
	)

	makeUserWithRanks := func(base string, artists []struct{ ID, Name string }) string {
		id := uuid.New()
		username := base + "_" + id.String()[:8]
		pw := "$2a$10$abcdefghijklmnopqrstuv"
		_, err := repo.CreateUser(ctx, id, username, username+"@test.com", "Overlap", "Tester", pw, "2Y", 2026)
		Expect(err).NotTo(HaveOccurred())
		items := make([]business.SpotifyTopArtist, 0, len(artists))
		for i, a := range artists {
			items = append(items, business.SpotifyTopArtist{Rank: int32(i + 1), SpotifyArtistID: a.ID, Name: a.Name})
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

	It("truncates overlaps while preserving ordering", func() {
		// Anchor with multiple overlaps
		anchor := makeUserWithRanks("anchor_ov", []struct{ ID, Name string }{{"a1", "a1"}, {"a2", "a2"}, {"a3", "a3"}, {"a4", "a4"}})
		// Create another user with all four overlaps (highest)
		_ = makeUserWithRanks("full_ov", []struct{ ID, Name string }{{"a1", "a1"}, {"a2", "a2"}, {"a3", "a3"}, {"a4", "a4"}})
		// Another with first three
		_ = makeUserWithRanks("tri_ov", []struct{ ID, Name string }{{"a1", "a1"}, {"a2", "a2"}, {"a3", "a3"}})
		// Another with two best ranks
		_ = makeUserWithRanks("bi_ov", []struct{ ID, Name string }{{"a1", "a1"}, {"a2", "a2"}})
		// Non-overlapping user
		_ = makeUserWithRanks("none_ov", []struct{ ID, Name string }{{"z1", "z1"}, {"z2", "z2"}})

		fullResp, err := svc.FindMusicMatches(ctx, generated.ArtistsRequest{Artists: []string{"ignored"}}, anchor, "medium_term", true, 10, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(fullResp.Code).To(Equal(200))
		fullMatches, _ := fullResp.Body.([]*generated.MatchUser)
		if len(fullMatches) == 0 {
			Skip("no matches to evaluate")
		}
		baseline := fullMatches[0].Overlaps
		if len(baseline) < 3 {
			Skip("need >=3 overlaps to test truncation")
		}

		limitedResp, err2 := svc.FindMusicMatches(ctx, generated.ArtistsRequest{Artists: []string{"ignored"}}, anchor, "medium_term", true, 10, 2)
		Expect(err2).NotTo(HaveOccurred())
		Expect(limitedResp.Code).To(Equal(200))
		limitedMatches, _ := limitedResp.Body.([]*generated.MatchUser)
		Expect(limitedMatches).NotTo(BeEmpty())
		limited := limitedMatches[0].Overlaps
		Expect(len(limited)).To(Equal(2))
		Expect(limited[0].Name).To(Equal(baseline[0].Name))
		Expect(limited[1].Name).To(Equal(baseline[1].Name))

		// Validate baseline ordering (anchor+other, anchor, other)
		isSorted := sort.SliceIsSorted(baseline, func(i, j int) bool {
			ai := baseline[i].AnchorRank + baseline[i].OtherRank
			aj := baseline[j].AnchorRank + baseline[j].OtherRank
			if ai != aj {
				return ai < aj
			}
			if baseline[i].AnchorRank != baseline[j].AnchorRank {
				return baseline[i].AnchorRank < baseline[j].AnchorRank
			}
			return baseline[i].OtherRank < baseline[j].OtherRank
		})
		Expect(isSorted).To(BeTrue())
	})
})
