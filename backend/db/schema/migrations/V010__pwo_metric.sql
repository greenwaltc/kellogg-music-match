-- speeds artist_id lookups when joining r1 and r2
CREATE INDEX IF NOT EXISTS user_artists_artist_user ON user_artists (artist_id, user_id);

-- speeds ordered scans for per-user ranks (if you frequently return lists)
CREATE INDEX IF NOT EXISTS user_artists_user_rank ON user_artists (user_id, rank);

-- ============================================================================
-- Position-weighted Overlap Metric (PWO)
-- ============================================================================

-- PWO similarity between two users' ranked artist lists.
-- alpha in (0,1]; returns [0,1].
-- Uses artist_id & rank directly from user_artists (no need to materialize arrays).
CREATE OR REPLACE FUNCTION pwo_similarity(alpha double precision, u1 uuid, u2 uuid)
RETURNS double precision
LANGUAGE sql
STABLE
AS $$
WITH r1 AS (
  SELECT artist_id, rank
  FROM user_artists
  WHERE user_id = u1
),
r2 AS (
  SELECT artist_id, rank
  FROM user_artists
  WHERE user_id = u2
),
inter AS (
  -- intersection with both ranks
  SELECT r1.rank AS r1, r2.rank AS r2
  FROM r1
  JOIN r2 USING (artist_id)
),
lens AS (
  SELECT (SELECT count(*) FROM r1) AS n1,
         (SELECT count(*) FROM r2) AS n2
),
m AS (
  SELECT LEAST(n1, n2) AS m FROM lens
),
num AS (
  -- numerator: sum_x alpha^(r1(x)-1) * alpha^(r2(x)-1)
  SELECT COALESCE(SUM(POWER(alpha, (r1 - 1)) * POWER(alpha, (r2 - 1))), 0.0) AS num
  FROM inter
),
den AS (
  -- denominator: sum_{i=1..m} alpha^{2(i-1)}
  SELECT
    CASE
      WHEN (SELECT m FROM m) = 0 THEN 0.0
      WHEN alpha = 1 THEN (SELECT m FROM m)::double precision
      ELSE (1 - POWER(alpha * alpha, (SELECT m FROM m))) / (1 - alpha * alpha)
    END AS den
)
SELECT CASE WHEN den.den = 0 THEN 0.0 ELSE num.num / den.den END
FROM num, den;
$$;

-- returns double precision, so sqlc will generate float64
CREATE OR REPLACE FUNCTION pwo_distance(alpha float8, u1 uuid, u2 uuid)
RETURNS double precision
LANGUAGE sql
STABLE
AS $$
  SELECT 1::float8 - pwo_similarity(alpha, u1, u2)
$$;

-- Harmonic PWO: sum_x (1/r1(x))*(1/r2(x)) / sum_{i=1..m} (1/i^2)
CREATE OR REPLACE FUNCTION pwo_similarity_harmonic(u1 uuid, u2 uuid)
RETURNS double precision
LANGUAGE sql
STABLE
AS $$
WITH r1 AS (SELECT artist_id, rank FROM user_artists WHERE user_id = u1),
     r2 AS (SELECT artist_id, rank FROM user_artists WHERE user_id = u2),
     inter AS (SELECT r1.rank AS r1, r2.rank AS r2 FROM r1 JOIN r2 USING (artist_id)),
     lens AS (SELECT (SELECT count(*) FROM r1) AS n1, (SELECT count(*) FROM r2) AS n2),
     m AS (SELECT LEAST(n1, n2) AS m FROM lens),
     num AS (SELECT COALESCE(SUM(1.0/r1 * 1.0/r2), 0.0) AS num FROM inter),
     den AS (
       SELECT COALESCE(SUM(1.0/(i*i)), 0.0) AS den
       FROM generate_series(1, (SELECT m FROM m)) AS s(i)
     )
SELECT CASE WHEN den.den = 0 THEN 0.0 ELSE num.num / den.den END
FROM num, den;
$$;