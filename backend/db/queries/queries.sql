-- name: CreateUser :one
INSERT INTO users (id, username, email, first_name, last_name, password_hash, program, graduation_year)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 LIMIT 1;

-- name: GetUserByUsernameWithPassword :one
SELECT id, username, email, first_name, last_name, password_hash, created_at, updated_at, program, graduation_year 
FROM users WHERE username = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: UserExistsByUsername :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = $1);

-- name: UserExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);

-- name: UpdateUser :one
UPDATE users 
SET first_name = $2, last_name = $3, email = $4
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetAllUsers :many
SELECT * FROM users ORDER BY created_at;

-- Artist queries
-- name: CreateArtist :one
INSERT INTO artists (name)
VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: GetArtistByName :one
SELECT * FROM artists WHERE name = $1 LIMIT 1;

-- name: GetArtistByID :one
SELECT * FROM artists WHERE id = $1 LIMIT 1;

-- name: GetAllArtists :many
SELECT * FROM artists ORDER BY name;

-- name: SearchArtists :many
SELECT * FROM artists 
WHERE LOWER(name) LIKE LOWER($1) 
ORDER BY 
  CASE 
    WHEN LOWER(name) = LOWER($2) THEN 1
    WHEN LOWER(name) LIKE LOWER($3) THEN 2
    ELSE 3
  END,
  LENGTH(name),
  name
LIMIT $4;

-- User-Artist relationship queries
-- name: AddUserArtist :exec
INSERT INTO user_artists (user_id, artist_id, rank)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, artist_id, rank) DO NOTHING;

-- name: RemoveUserArtist :exec
DELETE FROM user_artists WHERE user_id = $1 AND artist_id = $2;

-- name: GetUserArtists :many
SELECT a.id, a.name, a.created_at
FROM artists a
JOIN user_artists ua ON a.id = ua.artist_id
WHERE ua.user_id = $1
ORDER BY a.name;

-- name: GetArtistUsers :many
SELECT u.id, u.username, u.email, u.first_name, u.last_name, u.created_at, u.updated_at
FROM users u
JOIN user_artists ua ON u.id = ua.user_id
WHERE ua.artist_id = $1
ORDER BY u.username;

-- name: ClearUserArtists :exec
DELETE FROM user_artists WHERE user_id = $1;

-- name: SetUserArtists :exec
WITH new_artists AS (
    INSERT INTO artists (name)
    SELECT unnest($2::text[])
    ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
    RETURNING id, name
),
ranked_artists AS (
    SELECT 
        a.id,
        a.name,
        ($3::int[])[array_position($2::text[], a.name)] as rank
    FROM new_artists a
)
INSERT INTO user_artists (user_id, artist_id, rank)
SELECT $1, id, rank FROM ranked_artists
ON CONFLICT (user_id, artist_id) DO UPDATE SET rank = EXCLUDED.rank;

-- Matching queries
-- name: GetUsersWithArtists :many
SELECT 
    u.id, u.username, u.email, u.first_name, u.last_name,
    COALESCE(array_agg(a.name) FILTER (WHERE a.name IS NOT NULL), '{}') as artist_names
FROM users u
LEFT JOIN user_artists ua ON u.id = ua.user_id
LEFT JOIN artists a ON ua.artist_id = a.id
GROUP BY u.id, u.username, u.email, u.first_name, u.last_name
ORDER BY u.username;

-- name: FindSimilarUsers :many
SELECT
  u1.username,
  u1.first_name,
  u1.last_name,
  u1.program,
  u1.graduation_year,
  (SELECT array_agg(a1.name ORDER BY ua1.rank ASC)
   FROM user_artists ua1
   JOIN artists a1 ON ua1.artist_id = a1.id
   WHERE ua1.user_id = u1.id) AS artists,
  spearman_distance(
    (SELECT array_agg(a1.name ORDER BY ua1.rank ASC)
     FROM user_artists ua1
     JOIN artists a1 ON ua1.artist_id = a1.id
     WHERE ua1.user_id = u1.id),
    (SELECT array_agg(a2.name ORDER BY ua2.rank ASC)
     FROM users u2
     JOIN user_artists ua2 ON u2.id = ua2.user_id
     JOIN artists a2 ON ua2.artist_id = a2.id
     WHERE u2.username = $1)
  ) AS distance
FROM users u1
WHERE u1.username != $1
  AND EXISTS (SELECT 1 FROM user_artists WHERE user_id = u1.id)
ORDER BY distance ASC
LIMIT 10;

-- Feedback queries
-- name: CreateFeedback :one
INSERT INTO feedback (user_id, feedback_text)
VALUES ($1, $2)
RETURNING *;

-- name: GetFeedbackByUser :many
SELECT * FROM feedback
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetAllFeedback :many
SELECT f.*, u.username, u.first_name, u.last_name
FROM feedback f
JOIN users u ON f.user_id = u.id
ORDER BY f.created_at DESC
LIMIT $1;