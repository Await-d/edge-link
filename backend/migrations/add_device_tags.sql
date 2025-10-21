-- Migration: Add tags field to devices table
-- Description: Add tags array field to support device tagging for rule matching

BEGIN;

-- Add tags column to devices table
ALTER TABLE devices
ADD COLUMN IF NOT EXISTS tags text[] DEFAULT '{}';

-- Create index on tags for faster matching
CREATE INDEX IF NOT EXISTS idx_devices_tags ON devices USING GIN (tags);

-- Add comment
COMMENT ON COLUMN devices.tags IS 'Device tags for rule matching and grouping';

COMMIT;
