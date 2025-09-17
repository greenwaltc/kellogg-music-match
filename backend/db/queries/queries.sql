-- name: CreateUser :one
INSERT INTO users (id, username, email, first_name, last_name, password_hash)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 LIMIT 1;

-- name: GetUserByUsernameWithPassword :one
SELECT id, username, email, first_name, last_name, password_hash, created_at, updated_at 
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
artist_mapping AS (
    SELECT 
        a.id, 
        ranks.rank,
        ROW_NUMBER() OVER () as row_num
    FROM new_artists a
    JOIN (SELECT unnest($3::int[]) as rank, generate_subscripts($3::int[], 1) as row_num) ranks
        ON generate_subscripts($2::text[], 1) = ranks.row_num
    WHERE a.name = ($2::text[])[ranks.row_num]
)
INSERT INTO user_artists (user_id, artist_id, rank)
SELECT $1, id, rank FROM artist_mapping
ON CONFLICT (user_id, artist_id, rank) DO NOTHING;

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
WITH user_artist_lists AS (
  -- Step 1: Create an ordered list of artist names for each user.
  SELECT
    u.id AS user_id,
    u.username,
    array_agg(a.name ORDER BY ua.rank ASC) AS artists
  FROM
    users u
  JOIN
    user_artists ua ON u.id = ua.user_id
  JOIN
    artists a ON ua.artist_id = a.id
  GROUP BY
    u.id, u.username
)
-- Step 2: Compare every user against the target user's list.
SELECT
  -- 'compare_user' is the user we are finding matches for.
  compare_user.username,
  -- 'target_user' is the one we are comparing against.
  spearman_distance(compare_user.artists, target_user.artists) AS distance
FROM
  user_artist_lists AS compare_user,
  user_artist_lists AS target_user
WHERE
  -- Define your target user here.
  target_user.username = $1
  -- Ensure we don't compare the user to themselves.
  AND compare_user.username != target_user.username
ORDER BY
  distance ASC
LIMIT 10;