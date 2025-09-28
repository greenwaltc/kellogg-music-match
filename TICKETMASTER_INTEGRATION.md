# Ticketmaster Integration

This document describes how the Ticketmaster API integration works in the Kellogg Music Match application.

## Configuration

The Ticketmaster integration uses the following environment variables:

### Required
- `TICKETMASTER_CONSUMER_KEY` - Your Ticketmaster API consumer key
- `TICKETMASTER_CONSUMER_SECRET` - Your Ticketmaster API consumer secret

### Optional (with defaults)
- `TICKETMASTER_BASE_URL` - API base URL (default: `https://app.ticketmaster.com/discovery/v2`)
- `TICKETMASTER_TIMEOUT` - HTTP timeout in seconds (default: `30`)
- `TICKETMASTER_MAX_RESULTS` - Max results per API call (default: `200`)
- `TICKETMASTER_DEFAULT_CITY` - Default city for searches (default: `Chicago`)
- `TICKETMASTER_DEFAULT_STATE` - Default state code (default: `IL`)
- `TICKETMASTER_DEFAULT_COUNTRY` - Default country code (default: `US`)

## Current Configuration

Your application is configured with:
- **Consumer Key**: `3RVuRqbo6iLpQj0iEG6UUAZiWa2Z5Y0O`
- **Consumer Secret**: `EzfZFlmQwTHXIrsb`
- **Default Location**: Chicago, IL, US

## Usage Examples

### Basic Concert Fetching

```go
// In your main application
cfg := config.Load()
concertService := business.NewConcertService(cfg)

// Validate configuration
if err := concertService.ValidateConfiguration(); err != nil {
    log.Printf("Concert service configuration error: %v", err)
    return
}

// Fetch upcoming concerts
ctx := context.Background()
concerts, err := concertService.GetUpcomingConcerts(ctx)
if err != nil {
    log.Printf("Error fetching concerts: %v", err)
    return
}

fmt.Printf("Found %d concerts\n", len(concerts.Embedded.Events))
```

### Artist-Specific Concerts

```go
// Fetch concerts for a specific artist
taylorConcerts, err := concertService.GetConcertsByArtist(ctx, "Taylor Swift")
if err != nil {
    log.Printf("Error fetching Taylor Swift concerts: %v", err)
    return
}

for _, event := range taylorConcerts.Embedded.Events {
    fmt.Printf("Concert: %s on %s\n", event.Name, event.Dates.Start.LocalDate)
}
```

## Deployment

### Docker Compose

The integration is automatically configured in `docker-compose.yml`:

```yaml
environment:
  TICKETMASTER_CONSUMER_KEY: 3RVuRqbo6iLpQj0iEG6UUAZiWa2Z5Y0O
  TICKETMASTER_CONSUMER_SECRET: EzfZFlmQwTHXIrsb
  TICKETMASTER_DEFAULT_CITY: Chicago
  # ... other config
```

### Kubernetes (Pulumi)

The Pulumi deployment includes all necessary environment variables for the Kubernetes pods.

## API Limits

- **Free Tier**: 5,000 API calls per day
- **Rate Limiting**: Ticketmaster enforces rate limits
- **Best Practices**: 
  - Cache results when possible
  - Use appropriate page sizes
  - Handle rate limit responses gracefully

## Security Notes

- API credentials are exposed in this example for development
- **For production**: Use Kubernetes secrets or environment-specific configuration
- Consider rotating API keys periodically

## Extending the Integration

### Adding New Features

1. **Venue-Specific Searches**: Add venue ID support to the proxy
2. **Genre Filtering**: Implement genre-based concert filtering
3. **Price Range Filtering**: Add price filtering capabilities
4. **User Preferences**: Integrate with user artist preferences

### Example: Adding Genre Support

```go
// In TicketmasterConfig
type TicketmasterConfig struct {
    // ... existing fields
    DefaultGenres []string
}

// In proxy method
func (p *TicketmasterProxy) FetchConcertsByGenre(ctx context.Context, genre string) (*TicketmasterResponse, error) {
    params := url.Values{}
    params.Set("apikey", p.config.ConsumerKey)
    params.Set("classificationName", "music")
    params.Set("genreName", genre)
    // ... rest of implementation
}
```

## Testing

To test the integration:

```bash
# Build and run with Docker Compose
docker-compose up -d --build

# Check logs for concert service initialization
docker logs kmm-backend | grep -i concert

# Test API call (if you add an endpoint)
curl http://localhost:8080/api/concerts/upcoming
```

## Troubleshooting

### Common Issues

1. **Invalid API Key**: Check that environment variables are set correctly
2. **Rate Limiting**: Implement retry logic with exponential backoff
3. **No Results**: Verify location parameters and date ranges
4. **Timeout Errors**: Increase `TICKETMASTER_TIMEOUT` value

### Debug Mode

Enable debug logging to see API requests:

```yaml
environment:
  DEBUG_ENABLED: true
```

## Future Integration Ideas

1. **User Concert Recommendations**: Based on artist preferences
2. **Social Features**: Share concerts with music-compatible matches  
3. **Calendar Integration**: Add concerts to user calendars
4. **Price Alerts**: Notify users of price drops
5. **Venue Recommendations**: Suggest venues based on user history