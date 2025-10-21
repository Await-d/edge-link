-- 创建密钥状态枚举
CREATE TYPE key_status_enum AS ENUM (
    'active',
    'pending_rotation',
    'revoked',
    'expired'
);

-- 创建 device_keys 表
CREATE TABLE IF NOT EXISTS device_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    status key_status_enum NOT NULL DEFAULT 'active',
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- 确保每个设备同一时间只有一个活跃密钥
    UNIQUE(device_id, status) WHERE status = 'active'
);

-- 创建索引
CREATE INDEX idx_device_keys_device_id ON device_keys(device_id);
CREATE INDEX idx_device_keys_status ON device_keys(status);
CREATE INDEX idx_device_keys_expires_at ON device_keys(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_device_keys_created_at ON device_keys(created_at);

-- 创建更新时间触发器
CREATE TRIGGER update_device_keys_updated_at
    BEFORE UPDATE ON device_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
