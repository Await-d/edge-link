DROP TRIGGER IF EXISTS update_diagnostic_bundles_updated_at ON diagnostic_bundles;
DROP INDEX IF EXISTS idx_diagnostic_bundles_device_id;
DROP INDEX IF EXISTS idx_diagnostic_bundles_status;
DROP INDEX IF EXISTS idx_diagnostic_bundles_requested_at;
DROP INDEX IF EXISTS idx_diagnostic_bundles_expires_at;
DROP TABLE IF EXISTS diagnostic_bundles;
DROP TYPE IF EXISTS diagnostic_status_enum;
