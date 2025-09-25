-- 0) Prep: small stats view + indexes (perf)

-- Distinct listener counts per artist
CREATE MATERIALIZED VIEW IF NOT EXISTS artist_listener_counts AS
SELECT ua.artist_id, COUNT(DISTINCT ua.user_id)::bigint AS n_listeners
FROM user_artists ua
GROUP BY ua.artist_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_alc_artist ON artist_listener_counts(artist_id);
REFRESH MATERIALIZED VIEW CONCURRENTLY artist_listener_counts;  -- run after bulk loads

-- Helpful indexes you mostly already have
CREATE INDEX IF NOT EXISTS idx_user_artists_artist_user ON user_artists(artist_id, user_id);
CREATE INDEX IF NOT EXISTS idx_user_artists_user_rank  ON user_artists(user_id, rank);

--------------------------------

-- 1) Artist↔Artist audience distance

-- 1A. Jaccard distance (intersection over union of listener sets)

CREATE OR REPLACE FUNCTION artist_jaccard_distance(a int, b int)
RETURNS double precision
LANGUAGE sql STABLE AS
$$
WITH a_u AS (SELECT DISTINCT user_id FROM user_artists WHERE artist_id = a),
     b_u AS (SELECT DISTINCT user_id FROM user_artists WHERE artist_id = b),
     inter AS (SELECT COUNT(*)::float8 AS c FROM a_u JOIN b_u USING (user_id)),
     uni AS (
       SELECT COUNT(*)::float8 AS c
       FROM (SELECT user_id FROM a_u UNION SELECT user_id FROM b_u) u
     )
SELECT
  CASE WHEN (SELECT c FROM uni) = 0 THEN 1.0
       ELSE 1.0 - (SELECT c FROM inter)/(SELECT c FROM uni)
  END;
$$;

-- 1B. Cosine distance over binary listener vectors (less sensitive to list size)

CREATE OR REPLACE FUNCTION artist_cosine_distance(a int, b int)
RETURNS double precision
LANGUAGE sql STABLE AS
$$
WITH a_u AS (SELECT DISTINCT user_id FROM user_artists WHERE artist_id = a),
     b_u AS (SELECT DISTINCT user_id FROM user_artists WHERE artist_id = b),
     inter AS (SELECT COUNT(*)::float8 AS c FROM a_u JOIN b_u USING (user_id)),
     na AS (SELECT COUNT(*)::float8 AS c FROM a_u),
     nb AS (SELECT COUNT(*)::float8 AS c FROM b_u)
SELECT
  CASE
    WHEN (SELECT c FROM na) = 0 OR (SELECT c FROM nb) = 0 THEN 1.0
    ELSE 1.0 - LEAST(1.0, (SELECT c FROM inter)/sqrt((SELECT c FROM na)*(SELECT c FROM nb)))
  END;
$$;

-- 2) Tiny MusicBrainz metadata distance (optional, lightweight)

-- This gives a small bonus when obvious metadata matches (type/country) agree and when life spans overlap. It’s only used when you have those fields.

CREATE OR REPLACE FUNCTION artist_musicbrainz_meta_distance(a int, b int)
RETURNS double precision
LANGUAGE sql STABLE AS
$$
WITH ab AS (
  SELECT
    a.artist_type AS at_a, b.artist_type AS at_b,
    a.country     AS c_a,  b.country     AS c_b,
    a.life_span_begin AS s_a, a.life_span_end AS e_a,
    b.life_span_begin AS s_b, b.life_span_end AS e_b
  FROM artists a, artists b
  WHERE a.id = $1 AND b.id = $2
),
scores AS (
  SELECT
    -- simple matches: 0 distance contribution when equal, small penalty when not/unknown
    (CASE WHEN at_a IS NOT NULL AND at_b IS NOT NULL AND at_a = at_b THEN 1 ELSE 0 END) AS same_type,
    (CASE WHEN c_a  IS NOT NULL AND c_b  IS NOT NULL AND c_a  = c_b  THEN 1 ELSE 0 END) AS same_country,
    -- temporal overlap score in [0,1] if we have both sides of spans; otherwise 0
    (CASE
       WHEN s_a IS NOT NULL AND s_b IS NOT NULL THEN
         -- approximate overlap using starts only (keeps it simple & available)
         (CASE
            WHEN e_a IS NOT NULL AND e_b IS NOT NULL THEN
              -- compute overlap of [s_a, e_a] and [s_b, e_b]; if negative, 0
              GREATEST(0.0,
                LEAST(e_a, e_b)::date - GREATEST(s_a, s_b)::date
              ) / NULLIF( (GREATEST(e_a, e_b)::date - LEAST(s_a, s_b)::date), 0)
            ELSE
              -- if no ends, compare starts: closer starts → closer artists
              1.0 / (1.0 + ABS(EXTRACT(day FROM (s_a - s_b))) )
          END)
       ELSE 0.0
     END)::float8 AS time_overlap
  FROM ab
),
sim AS (
  -- weighted similarity in [0,1]
  SELECT (0.5*same_type + 0.3*same_country + 0.2*LEAST(1.0, time_overlap)) AS s FROM scores
)
SELECT 1.0 - (SELECT s FROM sim);  -- distance
$$;

-- This intentionally has small weight (see blend below) so it nudges metal-with-metal, etc., but won’t override audience signals.

-- 3) Blended artist distance (recommended)

-- 3a. Uncached version

-- Blend audience Jaccard + cosine, then sprinkle in the MB metadata bonus when available.

CREATE OR REPLACE FUNCTION artist_distance_uncached(a int, b int)
RETURNS double precision
LANGUAGE sql STABLE
AS $$
WITH comp AS (
  SELECT
    artist_jaccard_distance(a,b) AS d_j,
    artist_cosine_distance(a,b)  AS d_c,
    -- may be NULL-ish if fields are missing; guard below
    NULLIF(artist_musicbrainz_meta_distance(a,b), NULL)::float8 AS d_mb
),
w AS (
  -- tune as you like; audience dominates
  SELECT 0.55::float8 AS wj, 0.35::float8 AS wc, 0.10::float8 AS wmb
),
usable AS (
  SELECT
    CASE WHEN d_j  IS NULL THEN 0 ELSE wj  END AS wj_,
    CASE WHEN d_c  IS NULL THEN 0 ELSE wc  END AS wc_,
    CASE WHEN d_mb IS NULL THEN 0 ELSE wmb END AS wmb_,
    COALESCE(d_j,  1.0) AS dj,
    COALESCE(d_c,  1.0) AS dc,
    COALESCE(d_mb, 1.0) AS dmb
  FROM comp, w
),
agg AS (
  SELECT (wj_+wc_+wmb_) AS ws, (wj_*dj + wc_*dc + wmb_*dmb) AS num FROM usable
)
SELECT CASE WHEN ws = 0 THEN 1.0 ELSE num/ws END FROM agg;
$$;

-- Returns [0,1] where 0 = identical, 1 = very different.
-- Convert to similarity on demand as 1 - artist_distance(a,b).

-- 3b. Cached wrapper that callers should use
-- Looks up (LEAST(a,b), GREATEST(a,b)).
-- If missing or stale (TTL), computes via artist_distance_uncached and upserts.
-- Marked VOLATILE so it can write.

-- TTL in minutes; set to NULL to disable staleness checks
CREATE OR REPLACE FUNCTION artist_distance(
  a_in int,
  b_in int,
  ttl_minutes int DEFAULT 10080  -- 7 days
)
RETURNS double precision
LANGUAGE plpgsql
VOLATILE
AS $$
DECLARE
  a int := LEAST(a_in, b_in);
  b int := GREATEST(a_in, b_in);
  d double precision;
  needs_refresh boolean := false;
BEGIN
  IF a = b THEN
    RETURN 0.0;
  END IF;

  -- Try cache (optionally enforce TTL)
  IF ttl_minutes IS NULL THEN
    SELECT distance INTO d
    FROM artist_neighbors
    WHERE a = a AND b = b;
  ELSE
    SELECT distance,
           (now() - updated_at > make_interval(mins => ttl_minutes)) AS expired
    INTO d, needs_refresh
    FROM artist_neighbors
    WHERE a = a AND b = b;

    IF d IS NOT NULL AND NOT needs_refresh THEN
      RETURN d;
    END IF;
  END IF;

  -- Miss or stale: compute fresh
  d := artist_distance_uncached(a, b);

  -- Best-effort write-through (ignore races/errors)
  BEGIN
    INSERT INTO artist_neighbors(a, b, distance, updated_at)
    VALUES (a, b, d, now())
    ON CONFLICT (a,b)
    DO UPDATE SET distance = EXCLUDED.distance, updated_at = now();
  EXCEPTION WHEN OTHERS THEN
    -- swallow write issues to keep read path reliable
    NULL;
  END;

  RETURN d;
END;
$$;

-- Your Chamfer / user-Chamfer functions can keep calling artist_distance(...) and will automatically benefit from caching.

-- 3c. (Nice to have) Automatic invalidation on data changes

-- If user_artists changes, cached distances involving that artist become stale. A simple row-level trigger that deletes affected pairs keeps things honest (your wrapper will recompute on next read).

CREATE OR REPLACE FUNCTION invalidate_artist_neighbors()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  -- For INSERT/DELETE/UPDATE of artist_id invalidate that artist’s rows
  IF TG_OP IN ('INSERT','UPDATE') THEN
    DELETE FROM artist_neighbors WHERE a = NEW.artist_id OR b = NEW.artist_id;
  END IF;
  IF TG_OP IN ('DELETE','UPDATE') THEN
    DELETE FROM artist_neighbors WHERE a = OLD.artist_id OR b = OLD.artist_id;
  END IF;
  RETURN NULL; -- statement-level invalidation; nothing to modify in the base row
END;
$$;

-- Invalidate when taste graph changes
DROP TRIGGER IF EXISTS trg_invalidate_artist_neighbors_ins ON user_artists;
DROP TRIGGER IF EXISTS trg_invalidate_artist_neighbors_del ON user_artists;
DROP TRIGGER IF EXISTS trg_invalidate_artist_neighbors_upd ON user_artists;

CREATE TRIGGER trg_invalidate_artist_neighbors_ins
AFTER INSERT ON user_artists
FOR EACH ROW EXECUTE FUNCTION invalidate_artist_neighbors();

CREATE TRIGGER trg_invalidate_artist_neighbors_del
AFTER DELETE ON user_artists
FOR EACH ROW EXECUTE FUNCTION invalidate_artist_neighbors();

CREATE TRIGGER trg_invalidate_artist_neighbors_upd
AFTER UPDATE OF artist_id, user_id, rank ON user_artists
FOR EACH ROW EXECUTE FUNCTION invalidate_artist_neighbors();

-- If that’s too aggressive for your write volume, switch to mark-stale (e.g., set updated_at = 'epoch') and let TTL refresh naturally, or batch invalidations with a nightly job.

-- 3d. Bulk warm-up / neighbor build (optional utility)

-- Precompute top-N neighbors for one seed artist
CREATE OR REPLACE FUNCTION build_artist_neighbors(seed int, n int DEFAULT 200)
RETURNS void
LANGUAGE sql
AS $$
INSERT INTO artist_neighbors(a,b,distance,updated_at)
SELECT LEAST(seed, b.id), GREATEST(seed, b.id),
       artist_distance_uncached(LEAST(seed, b.id), GREATEST(seed, b.id)),
       now()
FROM artists b
WHERE b.id <> seed
ORDER BY 3
LIMIT n
ON CONFLICT (a,b) DO UPDATE
SET distance = EXCLUDED.distance, updated_at = now();
$$;


-- 4) Set↔Set similarity (Chamfer)

-- 4A. Generic Chamfer on arrays of artist IDs

CREATE OR REPLACE FUNCTION chamfer_distance_artists(s1 int[], s2 int[])
RETURNS double precision
LANGUAGE sql STABLE
AS $$
WITH s1a AS (SELECT unnest(s1) AS a),
     s2a AS (SELECT unnest(s2) AS b),
     d1 AS (  -- directed S1 → S2
       SELECT AVG(nn.d) AS avg_d
       FROM s1a
       CROSS JOIN LATERAL (
         SELECT artist_distance(s1a.a, s2a.b) AS d
         FROM s2a
         ORDER BY d
         LIMIT 1
       ) nn
     ),
     d2 AS (  -- directed S2 → S1
       SELECT AVG(nn.d) AS avg_d
       FROM s2a
       CROSS JOIN LATERAL (
         SELECT artist_distance(s2a.b, s1a.a) AS d
         FROM s1a
         ORDER BY d
         LIMIT 1
       ) nn
     )
SELECT ((COALESCE((SELECT avg_d FROM d1), 1.0) + COALESCE((SELECT avg_d FROM d2), 1.0))/2.0);
$$;

CREATE OR REPLACE FUNCTION chamfer_similarity_artists(s1 int[], s2 int[])
RETURNS double precision
LANGUAGE sql STABLE
AS $$
SELECT 1.0 - chamfer_distance_artists(s1, s2);
$$;


-- 4B. Chamfer on two users’ top-K lists (leverages your rank)

-- This pulls each user’s top-K artists (by rank), applies an optional exponential decay that makes high-rank items “count more,” then runs Chamfer.

-- alpha in (0,1]; smaller alpha → stronger emphasis on top ranks
CREATE OR REPLACE FUNCTION user_chamfer_distance(u1 uuid, u2 uuid, top_k int DEFAULT 50, alpha float8 DEFAULT 1.0)
RETURNS double precision
LANGUAGE sql STABLE
AS $$
WITH r1 AS (
  SELECT artist_id, rank, POWER(alpha, rank-1) AS w
  FROM user_artists
  WHERE user_id = u1
  ORDER BY rank
  LIMIT top_k
),
r2 AS (
  SELECT artist_id, rank, POWER(alpha, rank-1) AS w
  FROM user_artists
  WHERE user_id = u2
  ORDER BY rank
  LIMIT top_k
),
-- S1 → S2, but weight points by the source user's rank weight
d1 AS (
  SELECT AVG(nn.d) AS avg_d
  FROM r1
  CROSS JOIN LATERAL (
    SELECT artist_distance(r1.artist_id, r2.artist_id) AS d
    FROM r2
    ORDER BY d
    LIMIT 1
  ) nn
),
-- S2 → S1
d2 AS (
  SELECT AVG(nn.d) AS avg_d
  FROM r2
  CROSS JOIN LATERAL (
    SELECT artist_distance(r2.artist_id, r1.artist_id) AS d
    FROM r1
    ORDER BY d
    LIMIT 1
  ) nn
)
SELECT ((COALESCE((SELECT avg_d FROM d1), 1.0) + COALESCE((SELECT avg_d FROM d2), 1.0))/2.0);
$$;

CREATE OR REPLACE FUNCTION user_chamfer_similarity(u1 uuid, u2 uuid, top_k int DEFAULT 50, alpha float8 DEFAULT 1.0)
RETURNS double precision
LANGUAGE sql STABLE
AS $$
SELECT 1.0 - user_chamfer_distance(u1,u2,top_k,alpha);
$$;

-- If you want the weighting to be explicitly applied to the averaging (not just via matching), replace AVG(nn.d) with SUM(w*nn.d)/SUM(w) inside d1 and d2, carrying w from the source set.

-- 5) Quick examples

-- -- Pairwise
-- SELECT a1.name, a2.name, 1.0 - artist_distance(a1.id, a2.id) AS similarity
-- FROM artists a1, artists a2
-- WHERE a1.name = 'Korn' AND a2.name = 'Slipknot';
-- 
-- -- Two hand-picked sets (by names)
-- WITH s1 AS (
--   SELECT array_agg(id) AS ids FROM artists WHERE name IN ('Korn','Slipknot','Deftones','Mudvayne')
-- ),
-- s2 AS (
--   SELECT array_agg(id) AS ids FROM artists WHERE name IN ('Taylor Swift','Billie Eilish','Olivia Rodrigo')
-- )
-- SELECT chamfer_similarity_artists((SELECT ids FROM s1),(SELECT ids FROM s2));
-- 
-- -- Two users' top 40 with rank emphasis
-- SELECT user_chamfer_similarity('00000000-0000-0000-0000-000000000001'::uuid,
--                                '00000000-0000-0000-0000-000000000002'::uuid,
--                                40, 0.85);
-- 

-- 6) Optional: cache hot pairs

-- For speed at query time, cache nearest neighbors:

-- Stores symmetric distances using (min(a,b), max(a,b)) as the key
CREATE TABLE IF NOT EXISTS artist_neighbors (
  a int NOT NULL,
  b int NOT NULL,
  distance double precision NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (a, b)
);

-- Helpful if you’ll run TTL checks in WHERE clauses
CREATE INDEX IF NOT EXISTS idx_artist_neighbors_updated_at
  ON artist_neighbors(updated_at);

-- Then have artist_distance(a,b) first SELECT distance FROM artist_neighbors and fall back to live compute when missing.


