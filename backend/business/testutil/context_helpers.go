package testutil

import "context"

// UserContext carries minimal user identity for tests.
type UserContext struct{ ID string }

func (u *UserContext) GetUserID() string { return u.ID }

// WithUser adds a user identity to context using the generic key expected by logger and API service fallback.
func WithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, any("user"), &UserContext{ID: userID})
}
