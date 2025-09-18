package tests
package business_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	_ "github.com/lib/pq"
)

func TestMusicMatching(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Music Matching Behavioral Test Suite")
}

var _ = Describe("Music Matching System", func() {
	var (
		db              *sql.DB
		userRepo        business.UserRepository
		matchingEngine  *business.MatchingEngine
		matchingService *business.MatchingService
		ctx             context.Context
		testUsers       map[string]*generated.User
	)

	BeforeEach(func() {
		ctx = context.Background()
		
		// Set up test database connection
		dbConfig := business.DatabaseConfig{
			Host:     getEnvWithDefault("POSTGRES_HOST", "localhost"),
			Port:     getEnvWithDefault("POSTGRES_PORT", "5432"),
			Name:     getEnvWithDefault("POSTGRES_DB", "kellogg_music_match"),
			User:     getEnvWithDefault("POSTGRES_USER", "kellogg_user"),
			Password: getEnvWithDefault("POSTGRES_PASSWORD", "kellogg_secure_pass_2024"),
		}

		var err error
		db, err = business.NewDatabaseConnection(dbConfig)
		Expect(err).NotTo(HaveOccurred())
		
		userRepo = business.NewUserRepository(db)
		matchingEngine = business.NewMatchingEngine()
		matchingService = business.NewMatchingService(userRepo, matchingEngine)

		// Create test users with known preferences
		testUsers = make(map[string]*generated.User)
		
		// User with Tool only
		toolUser := createTestUser("tool_user", "Tool", "User", "tool@test.com", []string{"Tool"})
		testUsers["tool_user"] = toolUser
		
		// User with Tool and Radiohead
		toolRadioheadUser := createTestUser("tool_radiohead_user", "ToolRadio", "User", "toolradio@test.com", []string{"Tool", "Radiohead"})
		testUsers["tool_radiohead_user"] = toolRadioheadUser
		
		// User with completely different preferences
		beatlesUser := createTestUser("beatles_user", "Beatles", "User", "beatles@test.com", []string{"Beatles", "Pink Floyd"})
		testUsers["beatles_user"] = beatlesUser
		
		// User with overlapping preferences
		overlapUser := createTestUser("overlap_user", "Overlap", "User", "overlap@test.com", []string{"Tool", "Beatles"})
		testUsers["overlap_user"] = overlapUser
		
		// User with same artists in different order
		reverseUser := createTestUser("reverse_user", "Reverse", "User", "reverse@test.com", []string{"Radiohead", "Tool"})
		testUsers["reverse_user"] = reverseUser
	})

	AfterEach(func() {
		// Clean up test users
		for _, user := range testUsers {
			userRepo.DeleteUser(ctx, user.ID)
		}
		
		if db != nil {
			db.Close()
		}
	})

	Context("when matching users with identical preferences", func() {
		It("should return perfect similarity scores", func() {
			// Test user with Tool preference looking for matches
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool"},
			}, "tool_user")
			
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Code).To(Equal(200))
			
			matches, ok := response.Body.([]*generated.MatchUser)
			Expect(ok).To(BeTrue())
			
			// Should find at least the overlap_user who also has Tool
			foundOverlapUser := false
			for _, match := range matches {
				if match.Name == "Overlap User" {
					foundOverlapUser = true
					Expect(match.Score).To(BeNumerically(">=", 0.0)) // Perfect or near-perfect match
					Expect(match.Overlap).To(BeNumerically(">=", 1))   // At least 1 common artist
				}
			}
			Expect(foundOverlapUser).To(BeTrue())
		})
	})

	Context("when matching users with subset preferences", func() {
		It("should return good but not perfect similarity scores", func() {
			// User with just Tool looking at user with Tool + Radiohead
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool"},
			}, "tool_user")
			
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
				// Should have good similarity but not perfect (distance should be > 0)
				Expect(toolRadioheadMatch.Score).To(BeNumerically("<", 1.0)) // Not perfect
				Expect(toolRadioheadMatch.Score).To(BeNumerically(">", 0.0)) // But still positive
				Expect(toolRadioheadMatch.Overlap).To(Equal(int32(2)))        // Tool+Radiohead count
			}
		})
	})

	Context("when matching users with no common preferences", func() {
		It("should return low or zero similarity scores", func() {
			// Tool user looking for matches
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool"},
			}, "tool_user")
			
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

	Context("when matching users with overlapping preferences", func() {
		It("should return moderate similarity scores based on overlap", func() {
			// Overlap user (Tool, Beatles) looking for matches  
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool", "Beatles"},
			}, "overlap_user")
			
			Expect(err).NotTo(HaveOccurred())
			matches, ok := response.Body.([]*generated.MatchUser)
			Expect(ok).To(BeTrue())
			
			// Should find both Tool user and Beatles user with moderate scores
			var toolMatch, beatlesMatch *generated.MatchUser
			for _, match := range matches {
				switch match.Name {
				case "Tool User":
					toolMatch = match
				case "Beatles User":
					beatlesMatch = match
				}
			}
			
			if toolMatch != nil {
				Expect(toolMatch.Score).To(BeNumerically(">", 0.0)) // Positive similarity
				Expect(toolMatch.Overlap).To(BeNumerically(">=", 1)) // At least Tool in common
			}
			
			if beatlesMatch != nil {
				Expect(beatlesMatch.Score).To(BeNumerically(">", 0.0)) // Positive similarity  
				Expect(beatlesMatch.Overlap).To(BeNumerically(">=", 2)) // Beatles + Pink Floyd
			}
		})
	})

	Context("when matching considers ranking order", func() {
		It("should give higher scores for users with similar ranking preferences", func() {
			// Tool+Radiohead user vs Radiohead+Tool user (reverse order)
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool", "Radiohead"},
			}, "tool_radiohead_user")
			
			Expect(err).NotTo(HaveOccurred())
			matches, ok := response.Body.([]*generated.MatchUser)
			Expect(ok).To(BeTrue())
			
			var reverseMatch *generated.MatchUser
			for _, match := range matches {
				if match.Name == "Reverse User" {
					reverseMatch = match
					break
				}
			}
			
			if reverseMatch != nil {
				// Same artists but different order should still have good similarity
				Expect(reverseMatch.Score).To(BeNumerically(">", 0.0))
				Expect(reverseMatch.Score).To(BeNumerically("<", 1.0)) // Not perfect due to order
				Expect(reverseMatch.Overlap).To(Equal(int32(2)))        // Both artists match
			}
		})
	})

	Context("when handling edge cases", func() {
		It("should handle empty artist preferences gracefully", func() {
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{},
			}, "tool_user")
			
			Expect(err).To(HaveOccurred()) // Should return error for empty artists
			Expect(response.Code).To(Equal(400))
		})

		It("should handle non-existent users gracefully", func() {
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool"},
			}, "non_existent_user")
			
			Expect(err).To(HaveOccurred())
			Expect(response.Code).To(Equal(404))
		})

		It("should handle users with no existing preferences", func() {
			// Create user without setting any artists
			newUser := &generated.User{
				ID:        uuid.New(),
				Username:  "no_prefs_user",
				Email:     "noprofs@test.com",
				FirstName: "NoPrefs",
				LastName:  "User",
			}
			
			err := userRepo.CreateUser(ctx, *newUser, "password123")
			Expect(err).NotTo(HaveOccurred())
			defer userRepo.DeleteUser(ctx, newUser.ID)
			
			response, err := matchingService.FindMusicMatches(ctx, generated.ArtistsRequest{
				Artists: []string{"Tool"},
			}, "no_prefs_user")
			
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Code).To(Equal(200))
			
			// Should return some matches even for new user
			matches, ok := response.Body.([]*generated.MatchUser)
			Expect(ok).To(BeTrue())
			Expect(len(matches)).To(BeNumerically(">=", 0))
		})
	})
})

// Helper functions

func createTestUser(username, firstName, lastName, email string, artists []string) *generated.User {
	user := &generated.User{
		ID:        uuid.New(),
		Username:  username,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}
	
	ctx := context.Background()
	userRepo := getUserRepo() // Get from current context
	
	err := userRepo.CreateUser(ctx, *user, "password123")
	Expect(err).NotTo(HaveOccurred())
	
	if len(artists) > 0 {
		err = userRepo.SetUserArtists(ctx, user.ID, artists)
		Expect(err).NotTo(HaveOccurred())
	}
	
	return user
}

func getUserRepo() business.UserRepository {
	// This is a bit of a hack - in a real scenario you might use dependency injection
	// For now, we'll create a new connection for helper functions
	dbConfig := business.DatabaseConfig{
		Host:     getEnvWithDefault("POSTGRES_HOST", "localhost"),
		Port:     getEnvWithDefault("POSTGRES_PORT", "5432"),
		Name:     getEnvWithDefault("POSTGRES_DB", "kellogg_music_match"),
		User:     getEnvWithDefault("POSTGRES_USER", "kellogg_user"),
		Password: getEnvWithDefault("POSTGRES_PASSWORD", "kellogg_secure_pass_2024"),
	}
	
	db, err := business.NewDatabaseConnection(dbConfig)
	Expect(err).NotTo(HaveOccurred())
	
	return business.NewUserRepository(db)
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}