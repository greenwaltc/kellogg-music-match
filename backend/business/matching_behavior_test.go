package business_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

var _ = Describe("Music Matching Engine Behavior", func() {
	var (
		matchingEngine *business.MatchingEngine
		testUsers      []*generated.User
	)

	BeforeEach(func() {
		matchingEngine = business.NewMatchingEngine()

		// Create test users with different preference patterns
		testUsers = []*generated.User{
			{
				Id:        uuid.New().String(),
				Username:  "identical_user",
				FirstName: "Identical",
				LastName:  "User",
				Email:     "identical@test.com",
				Artists:   []string{"Tool", "Radiohead", "Pink Floyd"},
			},
			{
				Id:        uuid.New().String(),
				Username:  "subset_user",
				FirstName: "Subset",
				LastName:  "User",
				Email:     "subset@test.com",
				Artists:   []string{"Tool"},
			},
			{
				Id:        uuid.New().String(),
				Username:  "superset_user",
				FirstName: "Superset",
				LastName:  "User",
				Email:     "superset@test.com",
				Artists:   []string{"Tool", "Radiohead", "Pink Floyd", "Beatles"},
			},
			{
				Id:        uuid.New().String(),
				Username:  "overlap_user",
				FirstName: "Overlap",
				LastName:  "User",
				Email:     "overlap@test.com",
				Artists:   []string{"Tool", "Beatles"},
			},
			{
				Id:        uuid.New().String(),
				Username:  "different_user",
				FirstName: "Different",
				LastName:  "User",
				Email:     "different@test.com",
				Artists:   []string{"Beatles", "Led Zeppelin"},
			},
			{
				Id:        uuid.New().String(),
				Username:  "empty_user",
				FirstName: "Empty",
				LastName:  "User",
				Email:     "empty@test.com",
				Artists:   []string{},
			},
		}
	})

	Context("when target user has identical preferences", func() {
		It("should return maximum similarity score for perfect matches", func() {
			targetArtists := []string{"Tool", "Radiohead", "Pink Floyd"}
			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", testUsers)

			// Find the identical user match
			var identicalMatch *generated.MatchUser
			for i := range matches {
				if matches[i].Name == "Identical User" {
					identicalMatch = &matches[i]
					break
				}
			}

			Expect(identicalMatch).NotTo(BeNil())
			Expect(identicalMatch.Score).To(BeNumerically(">=", 0.9)) // Near perfect Jaccard similarity
			Expect(identicalMatch.Overlap).To(Equal(int32(3)))        // All 3 artists match
		})
	})

	Context("when target user has subset preferences", func() {
		It("should return good similarity for subset matches", func() {
			targetArtists := []string{"Tool"}
			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", testUsers)

			// Should find multiple users with Tool in their preferences
			Expect(len(matches)).To(BeNumerically(">", 0))

			// Find specific matches
			var identicalMatch, supersetMatch, overlapMatch *generated.MatchUser
			for i := range matches {
				switch matches[i].Name {
				case "Identical User":
					identicalMatch = &matches[i]
				case "Superset User":
					supersetMatch = &matches[i]
				case "Overlap User":
					overlapMatch = &matches[i]
				}
			}

			// All should have positive scores since they contain Tool
			if identicalMatch != nil {
				Expect(identicalMatch.Score).To(BeNumerically(">", 0))
				Expect(identicalMatch.Overlap).To(Equal(int32(1)))
			}
			if supersetMatch != nil {
				Expect(supersetMatch.Score).To(BeNumerically(">", 0))
				Expect(supersetMatch.Overlap).To(Equal(int32(1)))
			}
			if overlapMatch != nil {
				Expect(overlapMatch.Score).To(BeNumerically(">", 0))
				Expect(overlapMatch.Overlap).To(Equal(int32(1)))
			}
		})
	})

	Context("when target user has no overlapping preferences", func() {
		It("should return no matches for completely different preferences", func() {
			targetArtists := []string{"Metallica", "Iron Maiden"}
			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", testUsers)

			// Should find no matches since none of our test users have these artists
			Expect(len(matches)).To(Equal(0))
		})
	})

	Context("when target user has partial overlapping preferences", func() {
		It("should return matches ordered by similarity score", func() {
			targetArtists := []string{"Tool", "Beatles"}
			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", testUsers)

			Expect(len(matches)).To(BeNumerically(">", 0))

			// Find the overlap user who has exactly these artists
			var overlapMatch *generated.MatchUser
			for i := range matches {
				if matches[i].Name == "Overlap User" {
					overlapMatch = &matches[i]
					break
				}
			}

			if overlapMatch != nil {
				Expect(overlapMatch.Score).To(BeNumerically(">=", 0.8)) // Should be high Jaccard similarity
				Expect(overlapMatch.Overlap).To(Equal(int32(2)))        // Both artists match
			}

			// Verify results are sorted by score (descending)
			for i := 1; i < len(matches); i++ {
				Expect(matches[i-1].Score).To(BeNumerically(">=", matches[i].Score))
			}
		})
	})

	Context("when handling edge cases", func() {
		It("should handle empty target artist list gracefully", func() {
			targetArtists := []string{}
			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", testUsers)

			// Should return no matches for empty target
			Expect(len(matches)).To(Equal(0))
		})

		It("should skip users with empty artist lists", func() {
			targetArtists := []string{"Tool"}
			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", testUsers)

			// Should not include the empty_user in results
			for i := range matches {
				Expect(matches[i].Name).NotTo(Equal("Empty User"))
			}
		})

		It("should skip the caller from results", func() {
			targetArtists := []string{"Tool"}
			matches := matchingEngine.ComputeMatches(targetArtists, "identical_user", testUsers)

			// Should not include the caller (identical_user) in results
			for i := range matches {
				Expect(matches[i].Name).NotTo(Equal("Identical User"))
			}
		})

		It("should handle whitespace and normalization", func() {
			// Create user with whitespace in artist names
			userWithWhitespace := &generated.User{
				Id:        uuid.New().String(),
				Username:  "whitespace_user",
				FirstName: "Whitespace",
				LastName:  "User",
				Email:     "whitespace@test.com",
				Artists:   []string{" Tool ", "radiohead", "PINK FLOYD"},
			}

			testUsersWithWhitespace := append(testUsers, userWithWhitespace)

			targetArtists := []string{"tool", "RADIOHEAD", " pink floyd "}
			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", testUsersWithWhitespace)

			// Should find the whitespace user despite case/whitespace differences
			var whitespaceMatch *generated.MatchUser
			for i := range matches {
				if matches[i].Name == "Whitespace User" {
					whitespaceMatch = &matches[i]
					break
				}
			}

			Expect(whitespaceMatch).NotTo(BeNil())
			Expect(whitespaceMatch.Overlap).To(Equal(int32(3))) // All 3 should match after normalization
		})
	})

	Context("when validating Jaccard similarity calculation", func() {
		It("should calculate correct Jaccard scores for known cases", func() {
			// Test case: target {Tool, Radiohead} vs user {Tool, Beatles}
			// Intersection: {Tool} = 1
			// Union: {Tool, Radiohead, Beatles} = 3
			// Jaccard = 1/3 ≈ 0.3333

			targetArtists := []string{"Tool", "Radiohead"}
			userWithOneOverlap := &generated.User{
				Id:        uuid.New().String(),
				Username:  "one_overlap_user",
				FirstName: "OneOverlap",
				LastName:  "User",
				Email:     "oneoverlap@test.com",
				Artists:   []string{"Tool", "Beatles"},
			}

			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", []*generated.User{userWithOneOverlap})

			Expect(len(matches)).To(Equal(1))
			Expect(matches[0].Overlap).To(Equal(int32(1)))
			Expect(matches[0].Score).To(BeNumerically("~", 0.3333, 0.01))
		})

		It("should handle perfect Jaccard similarity", func() {
			// Test case: identical sets should give Jaccard = 1.0
			targetArtists := []string{"Tool", "Radiohead"}
			userIdentical := &generated.User{
				Id:        uuid.New().String(),
				Username:  "identical_two_user",
				FirstName: "IdenticalTwo",
				LastName:  "User",
				Email:     "identicaltwo@test.com",
				Artists:   []string{"Tool", "Radiohead"},
			}

			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", []*generated.User{userIdentical})

			Expect(len(matches)).To(Equal(1))
			Expect(matches[0].Overlap).To(Equal(int32(2)))
			Expect(matches[0].Score).To(BeNumerically("~", 1.0, 0.001))
		})
	})
})

// Test to validate the algorithm behavior that matches database function expectations
var _ = Describe("Algorithm Validation Against Database Distance Function", func() {
	var matchingEngine *business.MatchingEngine

	BeforeEach(func() {
		matchingEngine = business.NewMatchingEngine()
	})

	Context("comparing with expected spearman_distance outcomes", func() {
		It("should produce high scores for cases that would give distance=0 in database", func() {
			// Identical preferences should produce maximum Jaccard similarity
			targetArtists := []string{"Tool", "Radiohead", "Pink Floyd"}
			identicalUser := &generated.User{
				Username: "identical",
				Artists:  []string{"Tool", "Radiohead", "Pink Floyd"},
			}

			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", []*generated.User{identicalUser})

			Expect(len(matches)).To(Equal(1))
			Expect(matches[0].Score).To(BeNumerically(">=", 0.95)) // Very high score
		})

		It("should produce moderate scores for cases that would give distance=0.7 in database", func() {
			// Subset relationships should produce moderate Jaccard similarity
			targetArtists := []string{"Tool"}
			supersetUser := &generated.User{
				Username: "superset",
				Artists:  []string{"Tool", "Radiohead"},
			}

			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", []*generated.User{supersetUser})

			Expect(len(matches)).To(Equal(1))
			// Jaccard for {Tool} vs {Tool, Radiohead} = 1/2 = 0.5
			Expect(matches[0].Score).To(BeNumerically("~", 0.5, 0.01))
		})

		It("should produce no matches for cases that would give distance=2.0 in database", func() {
			// No overlap should produce no matches in the engine
			targetArtists := []string{"Tool", "Radiohead"}
			differentUser := &generated.User{
				Username: "different",
				Artists:  []string{"Beatles", "Pink Floyd"},
			}

			matches := matchingEngine.ComputeMatches(targetArtists, "test_caller", []*generated.User{differentUser})

			// No matches since there's no overlap (engine filters out zero-overlap cases)
			Expect(len(matches)).To(Equal(0))
		})
	})
})
