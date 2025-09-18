-- Migration to update artist name limit to 240 characters
-- This ensures consistency with the frontend character limit

-- First, check if there are any existing artists longer than 240 characters
-- (This query is for information only, will be removed in production)
-- SELECT name, length(name) FROM artists WHERE length(name) > 240;

-- Update the artists table to enforce 240 character limit
ALTER TABLE artists ALTER COLUMN name TYPE VARCHAR(240);

-- Add constraint to ensure artist names don't exceed 240 characters
ALTER TABLE artists ADD CONSTRAINT check_artist_name_length CHECK (char_length(name) <= 240);