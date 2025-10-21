DROP TRIGGER IF EXISTS update_pre_shared_keys_updated_at ON pre_shared_keys;
DROP INDEX IF EXISTS idx_pre_shared_keys_organization_id;
DROP INDEX IF EXISTS idx_pre_shared_keys_key_hash;
DROP INDEX IF EXISTS idx_pre_shared_keys_expires_at;
DROP TABLE IF EXISTS pre_shared_keys;
