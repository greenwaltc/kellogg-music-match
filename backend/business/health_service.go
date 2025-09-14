package business

import (
	"context"
	"net/http"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// HealthService implements the business logic for health checks
type HealthService struct {
}

// NewHealthService creates a new health service
func NewHealthService() *HealthService {
	return &HealthService{}
}

// GetHealth implements health check business logic
func (s *HealthService) GetHealth(ctx context.Context) (generated.ImplResponse, error) {
	response := generated.GetHealth200Response{
		Status: "healthy",
		Time:   time.Now(),
	}
	return generated.Response(http.StatusOK, response), nil
}
