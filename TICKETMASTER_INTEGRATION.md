# Ticketmaster Integration

This document describes how the Ticketmaster API integration works in the Affyne application, including the Chicago Events feature with automated synchronization and UI components.

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
- `TICKETMASTER_DATE_RANGE_MONTHS` - Event date range in months (default: `12`)
- `TICKETMASTER_GEO_LATLONG` - Optional "lat,long" to search by geocoordinates (e.g. `41.8781,-87.6298`)
- `TICKETMASTER_RADIUS` - Radius for geo search (default: `0`, ignored when geo not set)
- `TICKETMASTER_RADIUS_UNIT` - `miles` (default) or `km`
- `TICKETMASTER_PAGE_DELAY_MS` - Delay between paginated requests in milliseconds (default: `250`)

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
  # Inter-page delay to respect TM rate-limits across pagination
  TICKETMASTER_PAGE_DELAY_MS: 250
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
docker logs affyne-backend | grep -i concert

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

## Chicago Events Feature

### Overview
The Chicago Events feature provides a complete concert discovery experience with:
- **6-month event range** (configurable via `TICKETMASTER_DATE_RANGE_MONTHS`)  
- **Automated sync** every 24 hours
- **950+ events** currently loaded (September 2025 - March 2026)
- **Artist search** with case-insensitive filtering
- **Infinite scroll** UI with efficient pagination

### API Endpoints

#### Get Chicago Events
```bash
# Basic event listing
GET /chicago/events?limit=20&offset=0

# Search by artist (case-insensitive)
GET /chicago/events?limit=20&offset=0&artist=taylor

# Response format
{
  "events": [...],
  "hasMore": true,
  "totalCount": 953
}
```

### Frontend Integration
The Angular Chicago Events component (`chicago-events.component.ts`) provides:
- **Infinite scroll** for efficient data loading
- **Real-time search** with debounced input (300ms)
- **Consistent theming** with application CSS variables
- **Responsive design** optimized for concert browsing
- **Ticket integration** with popup blocker handling

### Sync Service
The backend sync service (`sync_service.go`) handles:
- **Scheduled sync** every 24 hours
- **Pagination** through Ticketmaster API (200 events per page)
- **Configuration-driven** date ranges and location
- **Error handling** with graceful degradation
- **Event cleanup** removes past events automatically

### Data Storage
Events are stored in PostgreSQL with:
- **Normalized structure** (events, venues, artists tables)
- **UPSERT operations** prevent duplicates
- **Indexes** for efficient querying by date and artist
- **Foreign keys** maintain data integrity

### Current Status
- **Total Events**: 953 active events
- **Date Range**: September 29, 2025 - March 28, 2026
- **Monthly Distribution**:
  - October 2025: 382 events
  - November 2025: 312 events  
  - December 2025: 155 events
  - January 2026: 20 events
  - February 2026: 39 events
  - March 2026: 32 events

### Management Commands
```bash
# Check event status
make events-status

# Trigger manual sync  
make events-sync

# Search events
make events-search ARTIST=john

# View sample events
make events-sample
```

## Future Integration Ideas

1. **User Concert Recommendations**: Based on PWO music compatibility scores
2. **Social Features**: Share concerts with music-compatible matches  
3. **Calendar Integration**: Add concerts to user calendars
4. **Price Alerts**: Notify users of ticket price drops
5. **Geographic Expansion**: Add support for other cities beyond Chicago
5. **Venue Recommendations**: Suggest venues based on user history