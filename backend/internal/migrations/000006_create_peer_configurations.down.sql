DROP TRIGGER IF EXISTS update_peer_configurations_updated_at ON peer_configurations;
DROP INDEX IF EXISTS idx_peer_configurations_device_id;
DROP INDEX IF EXISTS idx_peer_configurations_peer_device_id;
DROP TABLE IF EXISTS peer_configurations;
