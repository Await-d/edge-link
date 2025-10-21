DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
DROP INDEX IF EXISTS idx_sessions_device_a_id;
DROP INDEX IF EXISTS idx_sessions_device_b_id;
DROP INDEX IF EXISTS idx_sessions_started_at;
DROP INDEX IF EXISTS idx_sessions_ended_at;
DROP TABLE IF EXISTS sessions;
DROP TYPE IF EXISTS connection_type_enum;
