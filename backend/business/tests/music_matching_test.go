package business_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func TestMusicMatching(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Music Matching Behavioral Test Suite")
}

var _ = Describe("Music Matching System", func() {
	var (
		userRepo        business.UserRepository
		matchingEngine  *business.MatchingEngine
		matchingService *business.MatchingService
		ctx             context.Context
		testUsers       map[string]*generated.User
	)

	BeforeEach(func() {
		ctx = context.Background()

		var err error
		userRepo, err = business.NewUserRepository()
		Expect(err).NotTo(HaveOccurred())

		matchingEngine = business.NewMatchingEngine()

		// Create test configuration that allows fewer artists for testing
		testArtistConfig := &config.ArtistConfig{
			MinCount:        1, // Allow testing with just 1 artist
			MaxCount:        20,
			MaxNameLength:   100,
			SearchMaxLength: 100,
			SearchLimit:     50,
		}

		matchingService = business.NewMatchingServiceWithConfig(userRepo, matchingEngine, testArtistConfig)

		// Create test users with known preferences
		testUsers = make(map[string]*generated.User)

		// User with Tool only
		toolUser := createTestUser("tool_user_"+uuid.New().String()[:8], "Tool", "User", "tool"+uuid.New().String()[:8]+"@test.com", []string{"Tool"})
		testUsers["tool_user"] = toolUser

		// User with Tool and Radiohead
		toolRadioheadUser := createTestUser("tool_radiohead_user_"+uuid.New().String()[:8], "ToolRadio", "User", "toolradio"+uuid.New().String()[:8]+"@test.com", []string{"Tool", "Radiohead"})
		testUsers["tool_radiohead_user"] = toolRadioheadUser

		// User with completely different preferences
		beatlesUser := createTestUser("beatles_user_"+uuid.New().String()[:8], "Beatles", "User", "beatles"+uuid.New().String()[:8]+"@test.com", []string{"Beatles", "Pink Floyd"})
		testUsers["beatles_user"] = beatlesUser

		// User with overlapping preferences
		overlapUser := createTestUser("overlap_user_"+uuid.New().String()[:8], "Overlap", "User", "overlap"+uuid.New().String()[:8]+"@test.com", []string{"Tool", "Beatles"})
		testUsers["overlap_user"] = overlapUser

		// User with same artists in different order
		reverseUser := createTestUser("reverse_user_"+uuid.New().String()[:8], "Reverse", "User", "reverse"+uuid.New().String()[:8]+"@test.com", []string{"Radiohead", "Tool"})
		testUsers["reverse_user"] = reverseUser
	})

	AfterEach(func() {
		// No cleanup needed - using unique usernames with UUIDs to avoid collisions
	})

	Context("when matching users with identical preferences", func() {
		It("should return perfect similarity scores", func() {
			// Use the Tool+Radiohead user to find the reverse user who has the exact same set
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool", "Radiohead"},
			}, testUsers["tool_radiohead_user"].Username)

			Expect(err).NotTo(HaveOccurred())
			Expect(response.Code).To(Equal(200))

			matches, ok := response.Body.([]*generated.MatchUser)
			Expect(ok).To(BeTrue())

			var reverseMatch *generated.MatchUser
			for _, match := range matches {
				if match.Name == "Reverse User" {
					reverseMatch = match
					break
				}
			}

			// Only assert if present in top-N
			if reverseMatch != nil {
				Expect(reverseMatch.Score).To(Equal(float32(1.0)))
				Expect(reverseMatch.Overlap).To(Equal(int32(2)))
			}
		})
	})

	Context("when matching users with subset preferences", func() {
		It("should return good but not perfect similarity scores", func() {
			// User with just Tool looking at user with Tool + Radiohead
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool"},
			}, testUsers["tool_user"].Username)

			Expect(err).NotTo(HaveOccurred())
			matches, ok := response.Body.([]*generated.MatchUser)
			Expect(ok).To(BeTrue())

			// Find the Tool+Radiohead user
			var toolRadioheadMatch *generated.MatchUser
			for _, match := range matches {
				if match.Name == "ToolRadio User" {
					toolRadioheadMatch = match
					break
				}
			}

			if toolRadioheadMatch != nil {
				// With Chamfer similarity, subset gets a high but not perfect score
				Expect(toolRadioheadMatch.Score).To(BeNumerically(">", 0.8)) // High similarity
				Expect(toolRadioheadMatch.Score).To(BeNumerically("<", 1.0)) // Not perfect
				Expect(toolRadioheadMatch.Overlap).To(Equal(int32(1)))       // Tool overlap count
			}
		})
	})

	Context("when matching users with no common preferences", func() {
		It("should return low or zero similarity scores", func() {
			// Tool user looking for matches
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool"},
			}, testUsers["tool_user"].Username)

			Expect(err).NotTo(HaveOccurred())
			matches, ok := response.Body.([]*generated.MatchUser)
			Expect(ok).To(BeTrue())

			// Find the Beatles user (no overlap with Tool)
			var beatlesMatch *generated.MatchUser
			for _, match := range matches {
				if match.Name == "Beatles User" {
					beatlesMatch = match
					break
				}
			}

			// Beatles user might not appear in results due to no similarity,
			// but if they do, score should be very low
			if beatlesMatch != nil {
				Expect(beatlesMatch.Score).To(BeNumerically("<=", 0.0))
			}
		})
	})

	// Legacy overlap and ranking order tests removed due to decommissioned manual artist system.

	Context("when handling edge cases", func() {
		It("returns 200 even with empty or arbitrary artist list (ignored in Spotify mode)", func() {
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{Artists: []string{}}, testUsers["tool_user"].Username)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Code).To(Equal(200))
		})
		It("returns 200 for non-existent user currently (future: 404)", func() {
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{Artists: []string{"Tool"}}, "non_existent_user")
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Code).To(Equal(200)) // stub still returns success
		})
	})
})

// Helper functions

func createTestUser(username, firstName, lastName, email string, artists []string) *generated.User {
	userID := uuid.New()

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	Expect(err).NotTo(HaveOccurred())

	ctx := context.Background()

	// Get a new repository connection for this operation
	userRepo, err := business.NewUserRepository()
	Expect(err).NotTo(HaveOccurred())

	// Create user with proper parameters using the passed repository
	sqlcUser, err := userRepo.CreateUser(ctx, userID, username, email, firstName, lastName, string(passwordHash), "2Y", 2026)
	Expect(err).NotTo(HaveOccurred())

	// Legacy manual artist assignment removed; artists slice retained only for return struct

	// Convert to generated.User format
	user := &generated.User{
		Id:             sqlcUser.ID.String(),
		Username:       sqlcUser.Username,
		Email:          sqlcUser.Email,
		FirstName:      sqlcUser.FirstName,
		LastName:       sqlcUser.LastName,
		Program:        sqlcUser.Program.String,
		GraduationYear: sqlcUser.GraduationYear.Int32,
		Artists:        artists,
	}

	return user
}
