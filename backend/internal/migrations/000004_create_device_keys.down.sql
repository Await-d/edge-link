DROP TRIGGER IF EXISTS update_device_keys_updated_at ON device_keys;
DROP INDEX IF EXISTS idx_device_keys_device_id;
DROP INDEX IF EXISTS idx_device_keys_status;
DROP INDEX IF EXISTS idx_device_keys_expires_at;
DROP INDEX IF EXISTS idx_device_keys_created_at;
DROP TABLE IF EXISTS device_keys;
DROP TYPE IF EXISTS key_status_enum;
