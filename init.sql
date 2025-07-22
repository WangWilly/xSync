-- PostgreSQL initialization script for xSync
-- This script creates the necessary extensions and initial setup

-- Create extensions if they don't exist
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Grant necessary permissions
GRANT ALL PRIVILEGES ON DATABASE xsync TO xsync;
