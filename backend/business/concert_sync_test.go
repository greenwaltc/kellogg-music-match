package business_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
)

var _ = Describe("Concert Synchronization", func() {
	var (
		cfg         *config.Config
		syncService *concert.SyncService
		mockRepo    concert.Repository
	)

	BeforeEach(func() {
		cfg = &config.Config{
			Database: config.DatabaseConfig{
				Host:     "localhost",
				Port:     "5432",
				Name:     "kellogg_music_match",
				User:     "kellogg_user",
				Password: "kellogg_password",
				SSLMode:  "disable",
			},
			Server: config.ServerConfig{
				Port: "8080",
			},
			Ticketmaster: config.TicketmasterConfig{
				ConsumerKey:    "test-consumer-key",
				ConsumerSecret: "test-consumer-secret",
				BaseURL:        "https://app.ticketmaster.com/discovery/v2",
				Timeout:        30,
				MaxResults:     200,
				DefaultCity:    "Chicago",
				DefaultState:   "IL",
				DefaultCountry: "US",
			},
		}

		// Initialize mocks
		mockProvider := &MockEventProvider{}
		mockRepo = concert.NewMockRepository()

		// Initialize sync service with mock dependencies
		syncService = concert.NewSyncService(mockProvider, mockRepo, cfg)
	})

	Describe("Service Initialization", func() {
		It("should create sync service successfully", func() {
			Expect(syncService).ToNot(BeNil())
		})

		It("should handle sync service startup", func() {
			// Test that we can start the sync service without errors
			ctx := context.Background()
			err := syncService.Start(ctx)
			defer syncService.Stop() // Clean up

			// Verify service started without error
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Manual Sync", func() {
		It("should trigger manual sync without errors", func() {
			ctx := context.Background()
			err := syncService.ManualSync(ctx)
			// With MockRepository, this should not error
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Configuration", func() {
		It("should have valid ticketmaster configuration", func() {
			Expect(cfg.Ticketmaster.ConsumerKey).To(Equal("test-consumer-key"))
			Expect(cfg.Ticketmaster.BaseURL).To(Equal("https://app.ticketmaster.com/discovery/v2"))
			Expect(cfg.Ticketmaster.MaxResults).To(Equal(200))
		})
	})

	Describe("Repository Integration", func() {
		It("should use repository interface", func() {
			// Test that the sync service can work with repository interface
			repo := concert.NewMockRepository()
			mockProvider := &MockEventProvider{}
			service := concert.NewSyncService(mockProvider, repo, cfg)
			Expect(service).ToNot(BeNil())
		})
	})

	Describe("Error Handling", func() {
		Context("when components are configured", func() {
			It("should handle sync status requests", func() {
				// This tests the sync status functionality
				ctx := context.Background()
				status, err := syncService.GetSyncStatus(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).ToNot(BeNil())
			})
		})
	})
})
