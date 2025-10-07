-- =======================
-- Users
-- =======================

-- name: CreateUser :one
INSERT INTO users (id, username, email, first_name, last_name, password_hash, program, graduation_year)
VALUES (sqlc.arg(id), sqlc.arg(username), sqlc.arg(email), sqlc.arg(first_name), sqlc.arg(last_name), sqlc.arg(password_hash), sqlc.arg(program), sqlc.arg(graduation_year))
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = sqlc.arg(username) LIMIT 1;

-- name: GetUserByUsernameWithPassword :one
SELECT * FROM users WHERE username = sqlc.arg(username) LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = sqlc.arg(email) LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = sqlc.arg(id) LIMIT 1;

-- name: UserExistsByUsername :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = sqlc.arg(username));

-- name: UserExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = sqlc.arg(email));

-- name: UpdateUser :one
UPDATE users 
SET first_name = sqlc.arg(first_name), last_name = sqlc.arg(last_name), email = sqlc.arg(email)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = sqlc.arg(id);

-- name: GetAllUsers :many
SELECT * FROM users ORDER BY created_at;

-- =======================
-- Matching helpers / reports
-- =======================

-- (Removed) GetUsersWithArtists: legacy manual artist relationship removed
-- Returning empty artist array per user now handled in application layer if needed

-- =======================
-- New similarity APIs (artist + set)
-- =======================

-- (Removed) PairwiseArtistSimilarityByNames: depends on legacy artists table
SELECT sqlc.arg(name1)::text AS name1, sqlc.arg(name2)::text AS name2, 0.0::float8 AS similarity;

-- (Removed) PairwiseArtistSimilarityByIDs: legacy artist_distance no longer applicable
SELECT 0.0::float8 AS similarity;

-- (Removed) ChamferSimilarityByNames: fallback returns 0
SELECT 0.0::float8 AS similarity;

-- (Removed) ChamferSimilarityByIDs
SELECT 0.0::float8 AS similarity;

-- (Removed) UserChamferSimilarity: legacy function
SELECT 0.0::float8 AS similarity;

-- =======================
-- Nearest neighbors (Chamfer-based)
-- Replace old PWO query with Chamfer under the same exported name
-- =======================

-- :username => the anchor profile
-- :limit_n  => how many matches to return
-- Simplified stub version now that legacy manual artist similarity is removed.
-- Returns other users with a constant distance (1.0 -> similarity 0) until Spotify-based
-- preference matching is implemented. Lim parameter controls max rows returned.
SELECT u.username,
       u.first_name,
       u.last_name,
       u.program,
       u.graduation_year,
       '{}'::text[] AS artists,
       1.0::float8 AS distance
FROM users u
WHERE u.username <> sqlc.arg(username)
ORDER BY u.created_at DESC
LIMIT sqlc.arg(lim);

-- (If you also want the :by-user-id variant, keep your FindSimilarUsersChamferByUserID below unchanged.)

-- Feedback queries
-- name: CreateFeedback :one
INSERT INTO feedback (user_id, feedback_text)
VALUES (sqlc.arg(user_id), sqlc.arg(feedback_text))
RETURNING *;

-- name: GetFeedbackByUser :many
SELECT * FROM feedback
WHERE user_id = sqlc.arg(user_id)
ORDER BY created_at DESC;

-- name: GetAllFeedback :many
SELECT f.*, u.username, u.first_name, u.last_name
FROM feedback f
JOIN users u ON f.user_id = u.id
ORDER BY f.created_at DESC
LIMIT sqlc.arg(lim);

-- =======================
-- Concert Events
-- =======================

-- name: UpsertVenue :one
INSERT INTO venues (id, name, street, city, state, country, postal, capacity)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(street), sqlc.arg(city), sqlc.arg(state), sqlc.arg(country), sqlc.arg(postal), sqlc.arg(capacity))
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    street = EXCLUDED.street,
    city = EXCLUDED.city,
    state = EXCLUDED.state,
    country = EXCLUDED.country,
    postal = EXCLUDED.postal,
    capacity = EXCLUDED.capacity,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpsertConcertArtist :one
INSERT INTO concert_artists (id, name, genres)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(genres))
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    genres = EXCLUDED.genres,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpsertConcertEvent :one
INSERT INTO concert_events (id, name, event_date, venue_id, genres, price_min, price_max, price_currency, ticket_url, description, status, age_restriction, provider, external_url)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(event_date), sqlc.arg(venue_id), sqlc.arg(genres), sqlc.arg(price_min), sqlc.arg(price_max), sqlc.arg(price_currency), sqlc.arg(ticket_url), sqlc.arg(description), sqlc.arg(status), sqlc.arg(age_restriction), sqlc.arg(provider), sqlc.arg(external_url))
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    event_date = EXCLUDED.event_date,
    venue_id = EXCLUDED.venue_id,
    genres = EXCLUDED.genres,
    price_min = EXCLUDED.price_min,
    price_max = EXCLUDED.price_max,
    price_currency = EXCLUDED.price_currency,
    ticket_url = EXCLUDED.ticket_url,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    age_restriction = EXCLUDED.age_restriction,
    provider = EXCLUDED.provider,
    external_url = EXCLUDED.external_url,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpsertEventArtist :exec
INSERT INTO concert_event_artists (event_id, artist_id, role)
VALUES (sqlc.arg(event_id), sqlc.arg(artist_id), sqlc.arg(role))
ON CONFLICT (event_id, artist_id) DO UPDATE SET
    role = EXCLUDED.role;

-- name: GetConcertEventByID :one
SELECT 
    ce.*,
    v.name as venue_name,
    v.street as venue_street, 
    v.city as venue_city,
    v.state as venue_state,
    v.country as venue_country,
    v.postal as venue_postal,
    v.capacity as venue_capacity
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
WHERE ce.id = sqlc.arg(id);

-- name: GetConcertEventsInDateRange :many
SELECT 
    ce.*,
    v.name as venue_name,
    v.street as venue_street,
    v.city as venue_city,
    v.state as venue_state,
    v.country as venue_country,
    v.postal as venue_postal,
    v.capacity as venue_capacity,
    /* Aggregated artist names */
    array_remove(array_agg(DISTINCT ca.name), NULL) AS artist_names,
    /* User interest buckets */
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'INTERESTED'), NULL) AS interested_user_ids,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'GOING'), NULL) AS going_user_ids,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'LOOKING_FOR_GROUP'), NULL) AS looking_for_group_user_ids
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
LEFT JOIN concert_event_artists cea ON ce.id = cea.event_id
LEFT JOIN concert_artists ca ON cea.artist_id = ca.id
LEFT JOIN user_concert_event_interest ucei ON ucei.event_id = ce.id
WHERE ce.event_date >= sqlc.arg(start_date)
  AND ce.event_date <= sqlc.arg(end_date)
  AND (sqlc.arg(city)::text IS NULL OR v.city ILIKE '%' || sqlc.arg(city) || '%')
  AND (sqlc.arg(status)::text IS NULL OR ce.status = sqlc.arg(status))
GROUP BY ce.id, v.name, v.street, v.city, v.state, v.country, v.postal, v.capacity
ORDER BY ce.event_date ASC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off_set);

-- name: UpsertUserConcertEventInterest :exec
INSERT INTO user_concert_event_interest (user_id, event_id, interest_status)
VALUES (sqlc.arg(user_id), sqlc.arg(event_id), sqlc.arg(interest_status))
ON CONFLICT (user_id, event_id)
DO UPDATE SET interest_status = EXCLUDED.interest_status, updated_at = NOW();

-- name: DeleteUserConcertEventInterest :exec
DELETE FROM user_concert_event_interest
WHERE user_id = sqlc.arg(user_id) AND event_id = sqlc.arg(event_id);

-- name: GetEventArtists :many
SELECT 
    ca.id,
    ca.name,
    ca.genres,
  cea.role,
  /* How many users have any interest in this event (all statuses) */
  (SELECT COUNT(DISTINCT ui.user_id) FROM user_concert_event_interest ui WHERE ui.event_id = cea.event_id) AS interested_user_count
FROM concert_artists ca
JOIN concert_event_artists cea ON ca.id = cea.artist_id
WHERE cea.event_id = sqlc.arg(event_id);

-- =======================
-- Spotify Tokens
-- =======================

-- name: UpsertSpotifyTokens :exec
INSERT INTO spotify_tokens (user_id, access_token, refresh_token_encrypted, expires_at, scope, token_type)
VALUES (sqlc.arg(user_id), sqlc.arg(access_token), sqlc.arg(refresh_token_encrypted), sqlc.arg(expires_at), sqlc.arg(scope), COALESCE(sqlc.arg(token_type), 'Bearer'))
ON CONFLICT (user_id) DO UPDATE SET
    access_token = EXCLUDED.access_token,
    refresh_token_encrypted = EXCLUDED.refresh_token_encrypted,
    expires_at = EXCLUDED.expires_at,
    scope = EXCLUDED.scope,
    token_type = EXCLUDED.token_type,
    updated_at = NOW();

-- name: GetSpotifyTokensByUser :one
SELECT * FROM spotify_tokens WHERE user_id = sqlc.arg(user_id) LIMIT 1;

-- =======================
-- Spotify Top Items
-- =======================

-- name: DeleteSpotifyTopArtistSnapshotForRange :exec
DELETE FROM spotify_top_artist_snapshots 
WHERE user_id = sqlc.arg(user_id) AND range = sqlc.arg(range) AND fetched_at < sqlc.arg(fetched_at_cutoff);

-- name: InsertSpotifyTopArtistSnapshot :exec
INSERT INTO spotify_top_artist_snapshots (user_id, fetched_at, range, item_rank, spotify_artist_id, name, genres, popularity, image_url)
VALUES (sqlc.arg(user_id), sqlc.arg(fetched_at), sqlc.arg(range), sqlc.arg(item_rank), sqlc.arg(spotify_artist_id), sqlc.arg(name), sqlc.arg(genres), sqlc.arg(popularity), sqlc.arg(image_url))
ON CONFLICT (user_id, range, item_rank) DO UPDATE SET
  spotify_artist_id = EXCLUDED.spotify_artist_id,
  name = EXCLUDED.name,
  genres = EXCLUDED.genres,
  popularity = EXCLUDED.popularity,
  image_url = EXCLUDED.image_url,
  fetched_at = EXCLUDED.fetched_at;

-- name: DeleteSpotifyTopTrackSnapshotForRange :exec
DELETE FROM spotify_top_track_snapshots 
WHERE user_id = sqlc.arg(user_id) AND range = sqlc.arg(range) AND fetched_at < sqlc.arg(fetched_at_cutoff);

-- name: InsertSpotifyTopTrackSnapshot :exec
INSERT INTO spotify_top_track_snapshots (user_id, fetched_at, range, item_rank, spotify_track_id, name, artist_names, artist_ids, album_name, album_id, popularity, preview_url, duration_ms, image_url)
VALUES (sqlc.arg(user_id), sqlc.arg(fetched_at), sqlc.arg(range), sqlc.arg(item_rank), sqlc.arg(spotify_track_id), sqlc.arg(name), sqlc.arg(artist_names), sqlc.arg(artist_ids), sqlc.arg(album_name), sqlc.arg(album_id), sqlc.arg(popularity), sqlc.arg(preview_url), sqlc.arg(duration_ms), sqlc.arg(image_url))
ON CONFLICT (user_id, range, item_rank) DO UPDATE SET
  spotify_track_id = EXCLUDED.spotify_track_id,
  name = EXCLUDED.name,
  artist_names = EXCLUDED.artist_names,
  artist_ids = EXCLUDED.artist_ids,
  album_name = EXCLUDED.album_name,
  album_id = EXCLUDED.album_id,
  popularity = EXCLUDED.popularity,
  preview_url = EXCLUDED.preview_url,
  duration_ms = EXCLUDED.duration_ms,
  image_url = EXCLUDED.image_url,
  fetched_at = EXCLUDED.fetched_at;


-- name: GetEventsForArtist :many
SELECT 
    ce.*,
    v.name as venue_name,
    v.street as venue_street,
    v.city as venue_city,
    v.state as venue_state,
    v.country as venue_country,
    v.postal as venue_postal,
    v.capacity as venue_capacity,
    ca.name as artist_name,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'INTERESTED'), NULL) AS interested_user_ids,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'GOING'), NULL) AS going_user_ids,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'LOOKING_FOR_GROUP'), NULL) AS looking_for_group_user_ids
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
JOIN concert_event_artists cea ON ce.id = cea.event_id
JOIN concert_artists ca ON cea.artist_id = ca.id
 LEFT JOIN user_concert_event_interest ucei ON ucei.event_id = ce.id
WHERE ca.name ILIKE '%' || sqlc.arg(artist_name) || '%'
  AND ce.event_date >= CURRENT_TIMESTAMP
  AND (sqlc.arg(city)::text IS NULL OR v.city ILIKE '%' || sqlc.arg(city) || '%')
GROUP BY ce.id, v.name, v.street, v.city, v.state, v.country, v.postal, v.capacity, ca.name
ORDER BY ce.event_date ASC
LIMIT sqlc.arg(lim);

-- name: DeleteOldConcertEvents :exec
DELETE FROM concert_events 
WHERE event_date < sqlc.arg(cutoff_date);

-- name: GetConcertEventCount :one
-- Optional interest_status filter: if provided (non-empty), counts events having at least one user with that status
SELECT COUNT(DISTINCT ce.id)
FROM concert_events ce
LEFT JOIN user_concert_event_interest ucei ON ucei.event_id = ce.id
WHERE (sqlc.arg(interest_status) = '' OR ucei.interest_status = sqlc.arg(interest_status) OR ucei.interest_status IS NULL);

-- name: GetUpcomingConcertEventsInCity :many
SELECT 
    ce.*,
    v.name as venue_name,
    v.street as venue_street,
    v.city as venue_city,
    v.state as venue_state,
    v.country as venue_country,
    v.postal as venue_postal,
    v.capacity as venue_capacity,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'INTERESTED'), NULL) AS interested_user_ids,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'GOING'), NULL) AS going_user_ids,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'LOOKING_FOR_GROUP'), NULL) AS looking_for_group_user_ids
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
LEFT JOIN user_concert_event_interest ucei ON ucei.event_id = ce.id
WHERE ce.event_date >= CURRENT_TIMESTAMP
  AND v.city ILIKE '%' || sqlc.arg(city) || '%'
  AND ce.status = 'onsale'
GROUP BY ce.id, v.name, v.street, v.city, v.state, v.country, v.postal, v.capacity
ORDER BY ce.event_date ASC
LIMIT sqlc.arg(lim);

-- name: GetChicagoEventsWithArtistSearch :many
SELECT 
    ce.*,
    v.name as venue_name,
    v.street as venue_street,
    v.city as venue_city,
    v.state as venue_state,
    v.country as venue_country,
    v.postal as venue_postal,
    v.capacity as venue_capacity,
  COALESCE(jsonb_agg(DISTINCT jsonb_build_object('id', ca.id, 'name', ca.name, 'genres', ca.genres)) FILTER (WHERE ca.id IS NOT NULL), '[]'::jsonb) AS artists_json,
  (array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'INTERESTED'), NULL))::text[] AS interested_user_ids,
  (array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'GOING'), NULL))::text[] AS going_user_ids,
  (array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'LOOKING_FOR_GROUP'), NULL))::text[] AS looking_for_group_user_ids
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
LEFT JOIN concert_event_artists cea ON ce.id = cea.event_id
LEFT JOIN concert_artists ca ON cea.artist_id = ca.id
 LEFT JOIN user_concert_event_interest ucei ON ucei.event_id = ce.id
WHERE ce.event_date >= CURRENT_TIMESTAMP
  AND v.city ILIKE '%Chicago%'
  AND ce.status = 'onsale'
  AND (sqlc.arg(artist_name) = '' OR ca.name ILIKE '%' || sqlc.arg(artist_name) || '%')
  AND (sqlc.arg(any_interest)::bool = false OR EXISTS (SELECT 1 FROM user_concert_event_interest u2 WHERE u2.event_id = ce.id))
GROUP BY ce.id, v.name, v.street, v.city, v.state, v.country, v.postal, v.capacity
ORDER BY ce.event_date ASC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: GetChicagoEventsCountWithArtistSearch :one
SELECT COUNT(DISTINCT ce.id)
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
LEFT JOIN concert_event_artists cea ON ce.id = cea.event_id
LEFT JOIN concert_artists ca ON cea.artist_id = ca.id
LEFT JOIN user_concert_event_interest ucei ON ucei.event_id = ce.id
WHERE ce.event_date >= CURRENT_TIMESTAMP
  AND v.city ILIKE '%Chicago%'
  AND ce.status = 'onsale'
  AND (sqlc.arg(artist_name) = '' OR ca.name ILIKE '%' || sqlc.arg(artist_name) || '%')
  AND (sqlc.arg(interest_status) = '' OR ucei.interest_status = sqlc.arg(interest_status) OR ucei.interest_status IS NULL)
  AND (sqlc.arg(any_interest)::bool = false OR EXISTS (SELECT 1 FROM user_concert_event_interest u2 WHERE u2.event_id = ce.id));

-- name: GetConcertEventsInDateRangeWithInterest :many
-- Returns events within date range plus associated venue, artists, and user interest buckets
SELECT 
    ce.*,
    v.name as venue_name,
    v.street as venue_street,
    v.city as venue_city,
    v.state as venue_state,
    v.country as venue_country,
    v.postal as venue_postal,
    v.capacity as venue_capacity,
    array_remove(array_agg(DISTINCT ca.name), NULL) AS artist_names,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'INTERESTED'), NULL) AS interested_user_ids,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'GOING'), NULL) AS going_user_ids,
  array_remove(array_agg(DISTINCT ucei.user_id::text) FILTER (WHERE ucei.interest_status = 'LOOKING_FOR_GROUP'), NULL) AS looking_for_group_user_ids
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
LEFT JOIN concert_event_artists cea ON ce.id = cea.event_id
LEFT JOIN concert_artists ca ON cea.artist_id = ca.id
LEFT JOIN user_concert_event_interest ucei ON ucei.event_id = ce.id
WHERE ce.event_date >= sqlc.arg(start_date)
  AND ce.event_date <= sqlc.arg(end_date)
  AND (sqlc.arg(city)::text IS NULL OR v.city ILIKE '%' || sqlc.arg(city) || '%')
  AND (sqlc.arg(status)::text IS NULL OR ce.status = sqlc.arg(status))
GROUP BY ce.id, v.name, v.street, v.city, v.state, v.country, v.postal, v.capacity
ORDER BY ce.event_date ASC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off_set);

-- =======================
-- Password Reset Tokens
-- =======================

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (user_id, token, expires_at)
VALUES (sqlc.arg(user_id), sqlc.arg(token), sqlc.arg(expires_at))
RETURNING *;

-- name: GetPasswordResetToken :one
SELECT * FROM password_reset_tokens 
WHERE token = sqlc.arg(token) 
  AND expires_at > NOW() 
  AND used = FALSE 
LIMIT 1;

-- name: MarkPasswordResetTokenAsUsed :exec
UPDATE password_reset_tokens 
SET used = TRUE 
WHERE token = sqlc.arg(token);

-- name: DeleteExpiredPasswordResetTokens :exec
DELETE FROM password_reset_tokens 
WHERE expires_at < NOW() - INTERVAL '1 hour';

-- name: DeleteUserPasswordResetTokens :exec
DELETE FROM password_reset_tokens 
WHERE user_id = sqlc.arg(user_id);

-- name: UpdateUserPassword :one
UPDATE users 
SET password_hash = sqlc.arg(password_hash), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, username, email;