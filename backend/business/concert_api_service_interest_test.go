package business

import (
	"testing"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
)

// Test that convertToAPIConcert includes new user name arrays while retaining deprecated ID arrays.
func TestConvertToAPIConcert_IncludesUserNameArrays(t *testing.T) {
	svc := &ConcertAPIService{}

	evt := concert.Event{
		ID:                     "evt-123",
		Name:                   "Sample Show",
		Date:                   time.Now(),
		Venue:                  concert.Venue{Name: "Test Venue", Address: concert.Address{City: "Chicago", Country: "US"}},
		Artists:                []concert.Artist{{Name: "Artist One"}},
		Genres:                 []string{"Rock"},
		InterestedUserIDs:      []string{"u1", "u2"},
		GoingUserIDs:           []string{"u3"},
		LookingForGroupUserIDs: []string{"u4"},
		InterestedUsers:        []string{"Alice Johnson", "Bob Stone"},
		GoingUsers:             []string{"Carol Lee"},
		LookingForGroupUsers:   []string{"Dave Kim"},
	}

	api := svc.convertToAPIConcert(evt)

	// Deprecated ID arrays retained
	if len(api.InterestedUserIds) != 2 || api.InterestedUserIds[0] != "u1" {
		t.Fatalf("expected deprecated interestedUserIds retained, got %#v", api.InterestedUserIds)
	}
	if len(api.GoingUserIds) != 1 || api.GoingUserIds[0] != "u3" {
		t.Fatalf("expected deprecated goingUserIds retained, got %#v", api.GoingUserIds)
	}
	if len(api.LookingForGroupUserIds) != 1 || api.LookingForGroupUserIds[0] != "u4" {
		t.Fatalf("expected deprecated lookingForGroupUserIds retained, got %#v", api.LookingForGroupUserIds)
	}

	// New name arrays populated
	if len(api.InterestedUsers) != 2 || api.InterestedUsers[0] != "Alice Johnson" {
		t.Fatalf("expected interestedUsers names, got %#v", api.InterestedUsers)
	}
	if len(api.GoingUsers) != 1 || api.GoingUsers[0] != "Carol Lee" {
		t.Fatalf("expected goingUsers names, got %#v", api.GoingUsers)
	}
	if len(api.LookingForGroupUsers) != 1 || api.LookingForGroupUsers[0] != "Dave Kim" {
		t.Fatalf("expected lookingForGroupUsers names, got %#v", api.LookingForGroupUsers)
	}
}
