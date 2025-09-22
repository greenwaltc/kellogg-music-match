-- Migration to add feedback table for user suggestions
-- Allows registered users to submit feedback and suggestions

-- Create feedback table
CREATE TABLE feedback (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feedback_text TEXT NOT NULL CHECK (char_length(feedback_text) <= 1000 AND char_length(feedback_text) > 0),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_feedback_user_id ON feedback(user_id);
CREATE INDEX idx_feedback_created_at ON feedback(created_at);

-- Add trigger to automatically update updated_at
CREATE TRIGGER update_feedback_updated_at BEFORE UPDATE ON feedback
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();