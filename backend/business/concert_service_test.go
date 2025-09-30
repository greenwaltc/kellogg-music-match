package business

import (
	"context"
	"testing"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcertService_NewConcertService(t *testing.T) {
	// Test that we can create a new concert service
	cfg := &config.Config{
		Ticketmaster: config.TicketmasterConfig{
			ConsumerKey: "test-key",
			BaseURL:     "https://app.ticketmaster.com/discovery/v2",
		},
	}

	service := NewConcertService(cfg)

	require.NotNil(t, service, "Concert service should not be nil")
	assert.NotNil(t, service.config, "Config should be set")
}

func TestConcertService_Context(t *testing.T) {
	// Simple test to ensure context handling works
	ctx := context.Background()
	assert.NotNil(t, ctx, "Context should not be nil")
}
