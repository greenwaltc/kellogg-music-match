package business_test

import (
	"context"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Concert System Tests", func() {
	var (
		cfg *config.Config
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		cfg = &config.Config{Ticketmaster: config.TicketmasterConfig{ConsumerKey: "test-key", ConsumerSecret: "test-secret", BaseURL: "https://app.ticketmaster.com/discovery/v2", Timeout: 30, MaxResults: 200, DefaultCity: "Chicago", DefaultState: "IL", DefaultCountry: "US"}}
	})

	Describe("Concert Service", func() {
		It("should create service with default provider", func() {
			service := business.NewConcertService(cfg)
			Expect(service).ToNot(BeNil())
			Expect(service.GetProviderName()).To(Equal("Ticketmaster"))
			Expect(service.ValidateConfiguration(ctx)).To(Succeed())
		})
	})

	Describe("Ticketmaster Adapter", func() {
		It("should implement EventProvider interface", func() {
			adapter := concert.NewTicketmasterAdapter(&cfg.Ticketmaster)
			Expect(adapter.GetProviderName()).To(Equal("Ticketmaster"))
		})
	})
})
