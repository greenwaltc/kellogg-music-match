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

-- name: ClearUserArtists :exec
DELETE FROM user_artists WHERE user_id = sqlc.arg(user_id);

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
SELECT 1.0 - artist_distance(sqlc.arg(artist1_id)::int, sqlc.arg(artist1_id)::int) AS similarity;

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
WITH target AS (
  SELECT id AS target_id, program AS target_program, graduation_year AS target_grad
  FROM users
  WHERE username = sqlc.arg(username)
),
base_list AS (
  SELECT 1
  FROM user_artists ua
  JOIN target t ON ua.user_id = t.target_id
  LIMIT 1
)
SELECT
  u.username,
  u.first_name,
  u.last_name,
  u.program,
  u.graduation_year,
  COALESCE((
    SELECT array_agg(a.name ORDER BY ua.rank)
    FROM user_artists ua
    JOIN artists a ON a.id = ua.artist_id
    WHERE ua.user_id = u.id
  ), '{}'::text[]) AS artists,
  s.d AS distance,
  (1.0 - s.d) AS similarity
FROM users u
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
  AND EXISTS (SELECT 1 FROM user_artists ua WHERE ua.user_id = u.id)
ORDER BY
  s.d ASC,
  (u.program = t.target_program) DESC,
  (u.graduation_year = t.target_grad) DESC
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