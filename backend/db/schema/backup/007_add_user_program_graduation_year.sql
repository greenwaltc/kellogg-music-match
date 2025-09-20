-- Add program and graduation year fields to users table
-- Migration: 007_add_user_program_graduation_year.sql

-- Add program field with enum-like constraint
ALTER TABLE users 
ADD COLUMN program VARCHAR(10) CHECK (program IN ('2Y', '1Y', 'MMM', 'MBAi', 'JD-MBA', 'MD-MBA', 'EWMBA'));

-- Add graduation year field with reasonable range constraint
ALTER TABLE users 
ADD COLUMN graduation_year INTEGER CHECK (graduation_year >= 2025 AND graduation_year <= 2030);

-- Create indexes for performance
CREATE INDEX idx_users_program ON users(program);
CREATE INDEX idx_users_graduation_year ON users(graduation_year);

-- Add comments for documentation
COMMENT ON COLUMN users.program IS 'MBA program type: 2Y, 1Y, MBAi, MMM, or EWMBA';
COMMENT ON COLUMN users.graduation_year IS 'Expected graduation year (current year to 2030)';