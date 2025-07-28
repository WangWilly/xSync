-- Add storage_saved column to user_links table
ALTER TABLE user_links ADD COLUMN storage_saved BOOLEAN NOT NULL DEFAULT FALSE;
