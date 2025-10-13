-- Migration: 031_rolling_graduation_year_constraint.sql
-- Purpose: Replace fixed graduation_year CHECK constraint (2025..2030)
--          with a rolling validation of [current year .. current year + 5]

-- 1) Drop any existing fixed CHECK constraint on users.graduation_year
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM   information_schema.table_constraints tc
        WHERE  tc.table_name = 'users'
        AND    tc.constraint_type = 'CHECK'
        AND    tc.constraint_name = 'users_graduation_year_check'
    ) THEN
        EXECUTE 'ALTER TABLE users DROP CONSTRAINT users_graduation_year_check';
    END IF;
END$$;

-- 2) Create trigger function to validate graduation_year against rolling window
CREATE OR REPLACE FUNCTION users_validate_graduation_year()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    yr INTEGER;
    y  INTEGER;
BEGIN
    yr := NEW.graduation_year;
    y  := EXTRACT(YEAR FROM CURRENT_DATE)::INT;

    IF yr IS NOT NULL AND (yr < y OR yr > y + 5) THEN
        RAISE EXCEPTION 'graduation_year % out of allowed range [% - %]', yr, y, y + 5
            USING ERRCODE = '23514'; -- check_violation
    END IF;
    RETURN NEW;
END;
$$;

-- 3) Create trigger to enforce validation on INSERT/UPDATE of graduation_year
DROP TRIGGER IF EXISTS trg_users_validate_graduation_year ON users;
CREATE TRIGGER trg_users_validate_graduation_year
BEFORE INSERT OR UPDATE OF graduation_year ON users
FOR EACH ROW
EXECUTE FUNCTION users_validate_graduation_year();

-- 4) Update column comment to reflect rolling window behavior
COMMENT ON COLUMN users.graduation_year IS 'Expected graduation year (current year to current year + 5)';
