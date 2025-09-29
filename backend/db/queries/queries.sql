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
-- Artists
-- =======================

-- name: CreateArtist :one
INSERT INTO artists (name)
VALUES (sqlc.arg(name))
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: GetArtistByName :one
SELECT * FROM artists WHERE name = sqlc.arg(name) LIMIT 1;

-- name: GetArtistByID :one
SELECT * FROM artists WHERE id = sqlc.arg(id) LIMIT 1;

-- name: GetAllArtists :many
SELECT * FROM artists ORDER BY name;

-- name: SearchArtists :many
SELECT * FROM artists 
WHERE LOWER(name) LIKE LOWER(sqlc.arg(search_term)) 
ORDER BY 
  CASE 
    WHEN LOWER(name) = LOWER(sqlc.arg(exact_match)) THEN 1
    WHEN LOWER(name) LIKE LOWER(sqlc.arg(partial_match)) THEN 2
    ELSE 3
  END,
  LENGTH(name),
  name
LIMIT sqlc.arg(lim);

-- =======================
-- User-Artist relations
-- =======================

-- FIX: conflict target must be a unique/PK constraint. Your table has PK (user_id, artist_id)
-- name: AddUserArtist :exec
INSERT INTO user_artists (user_id, artist_id, rank)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(artist_id)::int, sqlc.arg(rank)::int)
ON CONFLICT (user_id, artist_id) DO UPDATE SET rank = EXCLUDED.rank;

-- name: RemoveUserArtist :exec
DELETE FROM user_artists WHERE user_id = sqlc.arg(user_id) AND artist_id = sqlc.arg(artist_id);

-- name: ClearUserArtists :exec
DELETE FROM user_artists WHERE user_id = sqlc.arg(user_id);

-- name: GetUserArtists :many
SELECT a.id, a.name, a.created_at
FROM artists a
JOIN user_artists ua ON a.id = ua.artist_id
WHERE ua.user_id = sqlc.arg(user_id)
ORDER BY ua.rank;

-- name: GetArtistUsers :many
SELECT u.id, u.username, u.email, u.first_name, u.last_name, u.created_at, u.updated_at
FROM users u
JOIN user_artists ua ON u.id = ua.user_id
WHERE ua.artist_id = sqlc.arg(artist_id)
ORDER BY u.username;

-- name: SetUserArtists :exec
WITH new_artists AS (
  INSERT INTO artists (name)
  SELECT unnest(sqlc.arg(artist_names)::text[])
  ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
  RETURNING id, name
),
 ranked_artists AS (
  SELECT 
    a.id,
    a.name,
    (sqlc.arg(ranks)::int[])[array_position(sqlc.arg(artist_names)::text[], a.name)] as rank
  FROM new_artists a
)
INSERT INTO user_artists (user_id, artist_id, rank)
SELECT sqlc.arg(user_id)::uuid, id, rank FROM ranked_artists
ON CONFLICT (user_id, artist_id) DO UPDATE SET rank = EXCLUDED.rank;

-- =======================
-- Matching helpers / reports
-- =======================

-- name: GetUsersWithArtists :many
SELECT 
    u.id, u.username, u.email, u.first_name, u.last_name,
    COALESCE(array_agg(a.name) FILTER (WHERE a.name IS NOT NULL), '{}') as artist_names
FROM users u
LEFT JOIN user_artists ua ON u.id = ua.user_id
LEFT JOIN artists a ON ua.artist_id = a.id
GROUP BY u.id, u.username, u.email, u.first_name, u.last_name
ORDER BY u.username;
-- =======================
-- New similarity APIs (artist + set)
-- =======================

-- name: PairwiseArtistSimilarityByNames :one
SELECT
  a1.name AS name1,
  a2.name AS name2,
  1.0 - artist_distance(a1.id, a2.id) AS similarity
FROM artists a1
JOIN artists a2 ON TRUE
WHERE a1.name = sqlc.arg(name1)
  AND a2.name = sqlc.arg(name2);

-- name: PairwiseArtistSimilarityByIDs :one
SELECT 1.0 - artist_distance(sqlc.arg(artist1_id)::int, sqlc.arg(artist2_id)::int) AS similarity;

-- name: ChamferSimilarityByNames :one
WITH s1 AS (
  SELECT COALESCE(array_agg(id), '{}'::int[]) AS ids
  FROM artists
  WHERE name = ANY(sqlc.arg(artist_names_list1)::text[])
),
s2 AS (
  SELECT COALESCE(array_agg(id), '{}'::int[]) AS ids
  FROM artists
  WHERE name = ANY(sqlc.arg(artist_names_list2)::text[])
)
SELECT chamfer_similarity_artists(
  (SELECT ids FROM s1),
  (SELECT ids FROM s2)
) AS similarity;

-- name: ChamferSimilarityByIDs :one
SELECT chamfer_similarity_artists(sqlc.arg(artist_ids_list1)::int[], sqlc.arg(artist_ids_list2)::int[]) AS similarity;

-- name: UserChamferSimilarity :one
SELECT user_chamfer_similarity(
  sqlc.arg(user1_id)::uuid,
  sqlc.arg(user2_id)::uuid,
  sqlc.arg(top_k)::int,
  sqlc.arg(alpha)::float8
) AS similarity;

-- =======================
-- Nearest neighbors (Chamfer-based)
-- Replace old PWO query with Chamfer under the same exported name
-- =======================

-- :username => the anchor profile
-- :limit_n  => how many matches to return
-- name: FindSimilarUsers :many
-- Performance optimizations:
-- 1) Restrict candidates to users who share at least 1 artist with the target (uses idx_user_artists_artist_user)
-- 2) Compute distances first and LIMIT to top-N before fetching per-user artist arrays
-- 3) Limit returned artist list to top_k via a lateral subquery to reduce memory/CPU
WITH target AS (
  SELECT id AS target_id, program AS target_program, graduation_year AS target_grad
  FROM users
  WHERE users.username = sqlc.arg(username)
),
base_list AS (
  -- ensure target has at least one artist; otherwise return no rows
  SELECT 1 FROM user_artists ua JOIN target t ON ua.user_id = t.target_id LIMIT 1
),
target_artists AS (
  SELECT ua.artist_id
  FROM user_artists ua
  JOIN target t ON ua.user_id = t.target_id
),
candidates AS (
  -- users with at least one overlapping artist with the target
  SELECT ua.user_id AS candidate_id
  FROM user_artists ua
  JOIN target_artists ta USING (artist_id)
  GROUP BY ua.user_id
),
scored AS (
  SELECT
    u.id,
    u.username,
    u.first_name,
    u.last_name,
    u.program,
    u.graduation_year,
    s.d AS distance
  FROM users u
  JOIN candidates c ON c.candidate_id = u.id
  JOIN target t ON TRUE
  CROSS JOIN LATERAL (
    SELECT user_chamfer_distance(
             u.id,
             t.target_id,
             sqlc.arg(top_k)::int,     -- top_k
             sqlc.arg(alpha)::float8   -- alpha
           ) AS d
  ) AS s
  WHERE u.username <> sqlc.arg(username)
    AND EXISTS (SELECT 1 FROM base_list)
  ORDER BY
    s.d ASC,
    (u.program = t.target_program) DESC,
    (u.graduation_year = t.target_grad) DESC
  LIMIT sqlc.arg(lim)
)
SELECT
  sc.username,
  sc.first_name,
  sc.last_name,
  sc.program,
  sc.graduation_year,
  COALESCE(al.artists, '{}'::text[]) AS artists,
  sc.distance AS distance,
  (1.0 - sc.distance)::int AS similarity
FROM scored sc
LEFT JOIN LATERAL (
  SELECT array_agg(sub.name) AS artists
  FROM (
    SELECT a.name
    FROM user_artists ua
    JOIN artists a ON a.id = ua.artist_id
    WHERE ua.user_id = sc.id
    ORDER BY ua.rank
    LIMIT sqlc.arg(top_k)::int
  ) sub
) al ON TRUE
ORDER BY
  sc.distance ASC,
  (sc.program = (SELECT target_program FROM target)) DESC,
  (sc.graduation_year = (SELECT target_grad FROM target)) DESC;

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
    v.capacity as venue_capacity
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
WHERE ce.event_date >= sqlc.arg(start_date)
  AND ce.event_date <= sqlc.arg(end_date)
  AND (sqlc.arg(city)::text IS NULL OR v.city ILIKE '%' || sqlc.arg(city) || '%')
  AND (sqlc.arg(status)::text IS NULL OR ce.status = sqlc.arg(status))
ORDER BY ce.event_date ASC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off_set);

-- name: GetEventArtists :many
SELECT 
    ca.id,
    ca.name,
    ca.genres,
    cea.role
FROM concert_artists ca
JOIN concert_event_artists cea ON ca.id = cea.artist_id
WHERE cea.event_id = sqlc.arg(event_id);

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
    ca.name as artist_name
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
JOIN concert_event_artists cea ON ce.id = cea.event_id
JOIN concert_artists ca ON cea.artist_id = ca.id
WHERE ca.name ILIKE '%' || sqlc.arg(artist_name) || '%'
  AND ce.event_date >= CURRENT_TIMESTAMP
  AND (sqlc.arg(city)::text IS NULL OR v.city ILIKE '%' || sqlc.arg(city) || '%')
ORDER BY ce.event_date ASC
LIMIT sqlc.arg(lim);

-- name: DeleteOldConcertEvents :exec
DELETE FROM concert_events 
WHERE event_date < sqlc.arg(cutoff_date);

-- name: GetConcertEventCount :one
SELECT COUNT(*) FROM concert_events;

-- name: GetUpcomingConcertEventsInCity :many
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
WHERE ce.event_date >= CURRENT_TIMESTAMP
  AND v.city ILIKE '%' || sqlc.arg(city) || '%'
  AND ce.status = 'onsale'
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
    v.capacity as venue_capacity
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
LEFT JOIN concert_event_artists cea ON ce.id = cea.event_id
LEFT JOIN concert_artists ca ON cea.artist_id = ca.id
WHERE ce.event_date >= CURRENT_TIMESTAMP
  AND v.city ILIKE '%Chicago%'
  AND ce.status = 'onsale'
  AND (sqlc.arg(artist_name) = '' OR ca.name ILIKE '%' || sqlc.arg(artist_name) || '%')
GROUP BY ce.id, v.name, v.street, v.city, v.state, v.country, v.postal, v.capacity
ORDER BY ce.event_date ASC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: GetChicagoEventsCountWithArtistSearch :one
SELECT COUNT(DISTINCT ce.id)
FROM concert_events ce
LEFT JOIN venues v ON ce.venue_id = v.id
LEFT JOIN concert_event_artists cea ON ce.id = cea.event_id
LEFT JOIN concert_artists ca ON cea.artist_id = ca.id
WHERE ce.event_date >= CURRENT_TIMESTAMP
  AND v.city ILIKE '%Chicago%'
  AND ce.status = 'onsale'
  AND (sqlc.arg(artist_name) = '' OR ca.name ILIKE '%' || sqlc.arg(artist_name) || '%');