-- name: CreateUser :one
INSERT INTO users (id, username, email, first_name, last_name, password_hash)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 LIMIT 1;

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
INSERT INTO user_artists (user_id, artist_id)
VALUES ($1, $2)
ON CONFLICT (user_id, artist_id) DO NOTHING;

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
    SELECT id FROM new_artists
    WHERE name = ANY($2::text[])
)
INSERT INTO user_artists (user_id, artist_id)
SELECT $1, id FROM artist_mapping
ON CONFLICT (user_id, artist_id) DO NOTHING;

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