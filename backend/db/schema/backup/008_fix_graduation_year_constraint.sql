-- Fix graduation year constraint to use fixed range instead of dynamic current year
-- Migration: 008_fix_graduation_year_constraint.sql

-- Drop the existing constraint (if it exists)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.constraint_column_usage 
        WHERE table_name = 'users' AND column_name = 'graduation_year'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_graduation_year_check;
    END IF;
END $$;

-- Add the corrected constraint with fixed range
ALTER TABLE users 
ADD CONSTRAINT users_graduation_year_check 
CHECK (graduation_year >= 2025 AND graduation_year <= 2030);