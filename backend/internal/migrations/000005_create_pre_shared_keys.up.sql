-- 创建 pre_shared_keys 表
CREATE TABLE IF NOT EXISTS pre_shared_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL UNIQUE,
    name VARCHAR(255),
    max_uses INTEGER DEFAULT NULL,
    used_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pre_shared_keys_organization_id ON pre_shared_keys(organization_id);
CREATE INDEX idx_pre_shared_keys_key_hash ON pre_shared_keys(key_hash);
CREATE INDEX idx_pre_shared_keys_expires_at ON pre_shared_keys(expires_at) WHERE expires_at IS NOT NULL;

CREATE TRIGGER update_pre_shared_keys_updated_at
    BEFORE UPDATE ON pre_shared_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
