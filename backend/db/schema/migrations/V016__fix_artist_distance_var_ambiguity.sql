-- Fix ambiguity in artist_distance due to variable names conflicting with table column names
-- Redefine function using distinct local variable names to avoid "column reference is ambiguous"

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
  a_key int := LEAST(a_in, b_in);
  b_key int := GREATEST(a_in, b_in);
  d double precision;
  needs_refresh boolean := false;
BEGIN
  IF a_key = b_key THEN
    RETURN 0.0;
  END IF;

  -- Try cache (optionally enforce TTL)
  IF ttl_minutes IS NULL THEN
    SELECT distance INTO d
    FROM artist_neighbors an
    WHERE an.a = a_key AND an.b = b_key;
  ELSE
    SELECT distance,
           (now() - updated_at > make_interval(mins => ttl_minutes)) AS expired
    INTO d, needs_refresh
    FROM artist_neighbors an
    WHERE an.a = a_key AND an.b = b_key;

    IF d IS NOT NULL AND NOT needs_refresh THEN
      RETURN d;
    END IF;
  END IF;

  -- Miss or stale: compute fresh
  d := artist_distance_uncached(a_key, b_key);

  -- Best-effort write-through (ignore races/errors)
  BEGIN
    INSERT INTO artist_neighbors(a, b, distance, updated_at)
    VALUES (a_key, b_key, d, now())
    ON CONFLICT (a,b)
    DO UPDATE SET distance = EXCLUDED.distance, updated_at = now();
  EXCEPTION WHEN OTHERS THEN
    -- swallow write issues to keep read path reliable
    NULL;
  END;

  RETURN d;
END;
$$;
