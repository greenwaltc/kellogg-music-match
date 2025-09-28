package business_test

import (
	"context"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestConcerts is handled by the main TestBusiness function in business_suite_test.go

var _ = Describe("Concert System Tests", func() {
	var (
		cfg *config.Config
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		cfg = &config.Config{
			Ticketmaster: config.TicketmasterConfig{
				ConsumerKey:    "test-key",
				ConsumerSecret: "test-secret",
				BaseURL:        "https://app.ticketmaster.com/discovery/v2",
				Timeout:        30,
				MaxResults:     200,
				DefaultCity:    "Chicago",
				DefaultState:   "IL",
				DefaultCountry: "US",
			},
		}
	})

	Describe("Concert Service", func() {
		It("should create service with default provider", func() {
			service := business.NewConcertService(cfg)
			Expect(service).ToNot(BeNil())
			Expect(service.GetProviderName()).To(Equal("Ticketmaster"))
		})

		It("should validate configuration", func() {
			service := business.NewConcertService(cfg)
			err := service.ValidateConfiguration(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject empty artist names in search", func() {
			service := business.NewConcertService(cfg)
			_, err := service.SearchEventsByArtist(ctx, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("artist name cannot be empty"))
		})

		It("should handle GetEventsForUser gracefully", func() {
			service := business.NewConcertService(cfg)
			result, err := service.GetEventsForUser(ctx, "test-user")

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Events).To(HaveLen(0))
		})

		Context("with invalid configuration", func() {
			It("should fail validation with missing consumer key", func() {
				invalidCfg := &config.Config{
					Ticketmaster: config.TicketmasterConfig{
						ConsumerSecret: "test-secret",
						BaseURL:        "https://app.ticketmaster.com/discovery/v2",
					},
				}

				service := business.NewConcertService(invalidCfg)
				err := service.ValidateConfiguration(ctx)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("consumer key not configured"))
			})
		})
	})

	Describe("Concert API Service", func() {
		It("should create API service successfully", func() {
			apiService := business.NewConcertAPIService(cfg)
			Expect(apiService).ToNot(BeNil())
		})

		It("should validate configuration", func() {
			apiService := business.NewConcertAPIService(cfg)
			err := apiService.ValidateConfiguration(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle empty event ID gracefully", func() {
			apiService := business.NewConcertAPIService(cfg)
			response, err := apiService.GetConcertById(ctx, "")

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Code).To(Equal(400))
		})

		It("should handle valid search parameters", func() {
			apiService := business.NewConcertAPIService(cfg)

			// This will return an error because we don't have real API credentials
			// but it should not crash
			response, err := apiService.SearchConcerts(ctx, "Taylor Swift", "Chicago", "IL", "US", "", "", "", 0, 20)

			Expect(err).ToNot(HaveOccurred())
			// Should return internal server error due to API call failure
			Expect(response.Code).To(Equal(500))
		})
	})

	Describe("Ticketmaster Adapter", func() {
		It("should implement EventProvider interface", func() {
			adapter := concert.NewTicketmasterAdapter(&cfg.Ticketmaster)

			var provider concert.EventProvider = adapter
			Expect(provider).ToNot(BeNil())
			Expect(provider.GetProviderName()).To(Equal("Ticketmaster"))
		})

		It("should validate configuration", func() {
			adapter := concert.NewTicketmasterAdapter(&cfg.Ticketmaster)
			err := adapter.IsHealthy(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail with missing configuration", func() {
			invalidCfg := &config.TicketmasterConfig{}
			adapter := concert.NewTicketmasterAdapter(invalidCfg)

			err := adapter.IsHealthy(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("consumer key not configured"))
		})
	})

	Describe("Provider Abstraction", func() {
		It("should allow custom provider injection", func() {
			// Create a mock provider
			mockProvider := &MockEventProvider{}

			service := business.NewConcertServiceWithProvider(mockProvider, cfg)
			Expect(service).ToNot(BeNil())
			Expect(service.GetProviderName()).To(Equal("MockProvider"))
		})
	})
})
