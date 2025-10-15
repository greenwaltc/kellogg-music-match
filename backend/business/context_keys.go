package business

// MatchBasisContextKey is a typed context key for passing the matching basis ("artists" or "tracks")
// between HTTP layer wrappers and the MatchingService business logic.
// Using a dedicated type avoids collisions and replaces the legacy string key "match_basis".
type MatchBasisContextKey struct{}

// MatchNameFilterContextKey is a typed context key for passing an optional fuzzy user name filter
// used to restrict similar users to those whose first/last name or username matches the provided query.
type MatchNameFilterContextKey struct{}
