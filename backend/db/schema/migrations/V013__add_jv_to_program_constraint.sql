-- Add 'JV' to program check constraint
-- Migration: V013__add_jv_to_program_constraint.sql

-- Drop the existing check constraint
ALTER TABLE users DROP CONSTRAINT users_program_check;

-- Add the updated constraint that includes 'JV'
ALTER TABLE users 
ADD CONSTRAINT users_program_check 
CHECK (program IN ('2Y', '1Y', 'MMM', 'MBAi', 'JD-MBA', 'MD-MBA', 'EWMBA', 'JV'));

-- Update the comment to match the actual constraint
COMMENT ON COLUMN users.program IS 'MBA program type: 2Y, 1Y, MMM, MBAi, JD-MBA, MD-MBA, EWMBA, JV';